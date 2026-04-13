package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
	k8sutil "github.com/vuldin/redpanda-check/internal/k8s"
)

// NoFractionalCPU validates CPU requests/limits are whole integers.
func NoFractionalCPU(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "no_fractional_cpu",
		Description: "No fractional CPU requests",
		Level:       checker.LevelCritical,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "Fractional CPU check only applicable to Kubernetes deployments"
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
			if !req.Cpu().IsZero() && req.Cpu().MilliValue()%1000 != 0 {
				podIssues = append(podIssues, fmt.Sprintf("request: %.2f cores", float64(req.Cpu().MilliValue())/1000.0))
			}
			if !lim.Cpu().IsZero() && lim.Cpu().MilliValue()%1000 != 0 {
				podIssues = append(podIssues, fmt.Sprintf("limit: %.2f cores", float64(lim.Cpu().MilliValue())/1000.0))
			}

			if len(podIssues) > 0 {
				issues = append(issues, fmt.Sprintf("%s: fractional CPU (%s)", pod.Name, strings.Join(podIssues, ", ")))
			} else if !req.Cpu().IsZero() || !lim.Cpu().IsZero() {
				good = append(good, fmt.Sprintf("%s: whole integer CPU", pod.Name))
			}
		}
	}

	if len(issues) > 0 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Fractional CPU detected:\n%s", strings.Join(issues, "\n"))
	} else if len(good) > 0 {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("All CPU requests/limits are whole integers:\n%s", strings.Join(good, "\n"))
	} else {
		r.Status = checker.StatusSkip
		r.Details = "No Redpanda containers found"
	}
	pc.AddResult(r)
}
