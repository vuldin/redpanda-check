package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
	k8sutil "github.com/vuldin/redpanda-check/internal/k8s"
)

// TuningInitContainer validates that the Redpanda tuning init container
// completed successfully on all pods. The Helm chart and operator run OS-level
// tuners (disk scheduler, IRQ affinity, etc.) as an init container. This check
// verifies those tuners ran without error.
func TuningInitContainer(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "tuning_init_container",
		Description: "Redpanda tuning init container completed",
		Level:       checker.LevelRecommended,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "Tuning init container check only applicable to Kubernetes deployments"
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

	if len(pods) == 0 {
		r.Status = checker.StatusSkip
		r.Details = "No Redpanda pods found"
		pc.AddResult(r)
		return
	}

	var failed []string
	var notFound []string
	succeeded := 0

	for _, pod := range pods {
		found := false
		for _, status := range pod.Status.InitContainerStatuses {
			if status.Name != "tuning" {
				continue
			}
			found = true
			terminated := status.State.Terminated
			if terminated != nil && terminated.ExitCode == 0 {
				succeeded++
			} else if terminated != nil {
				failed = append(failed, fmt.Sprintf("%s (exit code %d: %s)",
					pod.Name, terminated.ExitCode, terminated.Reason))
			} else {
				failed = append(failed, fmt.Sprintf("%s (not terminated)", pod.Name))
			}
			break
		}
		if !found {
			notFound = append(notFound, pod.Name)
		}
	}

	if len(failed) > 0 {
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("Tuning init container failed on: %s", strings.Join(failed, ", "))
	} else if len(notFound) == len(pods) {
		r.Status = checker.StatusSkip
		r.Details = "No tuning init container found on pods (may not be configured)"
	} else if len(notFound) > 0 {
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("Tuning init container missing on: %s; succeeded on %d pods",
			strings.Join(notFound, ", "), succeeded)
	} else {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("Tuning init container completed on all %d pods", succeeded)
	}
	pc.AddResult(r)
}
