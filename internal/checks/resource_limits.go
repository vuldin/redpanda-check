package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
	k8sutil "github.com/vuldin/redpanda-check/internal/k8s"
)

// ResourceLimits validates CPU and memory resource limits match requests.
func ResourceLimits(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "resource_limits",
		Description: "CPU and memory resource limits (requests = limits)",
		Level:       checker.LevelCritical,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "Resource limits check only applicable to Kubernetes deployments"
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
			lim := c.Resources.Limits

			var podIssues []string
			if req.Cpu().IsZero() {
				podIssues = append(podIssues, "no CPU request")
			}
			if req.Memory().IsZero() {
				podIssues = append(podIssues, "no memory request")
			}
			if lim.Cpu().IsZero() {
				podIssues = append(podIssues, "no CPU limit")
			}
			if lim.Memory().IsZero() {
				podIssues = append(podIssues, "no memory limit")
			}
			if !req.Cpu().IsZero() && !lim.Cpu().IsZero() && req.Cpu().Cmp(*lim.Cpu()) != 0 {
				podIssues = append(podIssues, fmt.Sprintf("CPU request (%s) != limit (%s)", req.Cpu(), lim.Cpu()))
			}
			if !req.Memory().IsZero() && !lim.Memory().IsZero() && req.Memory().Cmp(*lim.Memory()) != 0 {
				podIssues = append(podIssues, fmt.Sprintf("memory request (%s) != limit (%s)", req.Memory(), lim.Memory()))
			}

			if len(podIssues) > 0 {
				issues = append(issues, fmt.Sprintf("%s: %s", pod.Name, strings.Join(podIssues, ", ")))
			} else {
				good = append(good, fmt.Sprintf("%s: CPU %s, Memory %s", pod.Name, req.Cpu(), req.Memory()))
			}
		}
	}

	if len(issues) > 0 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Resource configuration issues:\n%s", strings.Join(issues, "\n"))
	} else if len(good) > 0 {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("Resource limits properly configured (requests = limits):\n%s", strings.Join(good, "\n"))
	} else {
		r.Status = checker.StatusSkip
		r.Details = "No Redpanda containers found in pods"
	}
	pc.AddResult(r)
}
