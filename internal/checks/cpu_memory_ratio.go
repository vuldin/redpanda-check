package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
	k8sutil "github.com/vuldin/redpanda-check/internal/k8s"
)

// CPUMemoryRatio validates the CPU to memory ratio is at least 1:2.
func CPUMemoryRatio(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "cpu_memory_ratio",
		Description: "CPU to memory ratio (1:2 minimum)",
		Level:       checker.LevelCritical,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "CPU to memory ratio check only applicable to Kubernetes deployments"
		pc.AddResult(r)
		return
	}

	pods, err := k8sutil.RedpandaPods(ctx, pc.K8sClient, pc.Namespace)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to list pods: %v", err)
		pc.AddResult(r)
		return
	}

	var issues, good []string
	for _, pod := range pods {
		for _, c := range pod.Spec.Containers {
			if c.Name != "redpanda" {
				continue
			}
			req := c.Resources.Requests
			if req.Cpu().IsZero() || req.Memory().IsZero() {
				issues = append(issues, fmt.Sprintf("%s: missing CPU or memory requests", pod.Name))
				continue
			}

			cpuCores := float64(req.Cpu().MilliValue()) / 1000.0
			memGiB := float64(req.Memory().Value()) / (1024 * 1024 * 1024)
			ratio := memGiB / cpuCores

			if ratio < 2.0 {
				issues = append(issues, fmt.Sprintf("%s: ratio %.1f:1 (CPU: %.1f cores, Memory: %.1f GiB)",
					pod.Name, ratio, cpuCores, memGiB))
			} else {
				good = append(good, fmt.Sprintf("%s: ratio %.1f:1 (CPU: %.1f cores, Memory: %.1f GiB)",
					pod.Name, ratio, cpuCores, memGiB))
			}
		}
	}

	if len(issues) > 0 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("CPU to memory ratio < 1:2 minimum:\n%s", strings.Join(issues, "\n"))
	} else if len(good) > 0 {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("CPU to memory ratio meets minimum 1:2:\n%s", strings.Join(good, "\n"))
	} else {
		r.Status = checker.StatusSkip
		r.Details = "No Redpanda containers found"
	}
	pc.AddResult(r)
}
