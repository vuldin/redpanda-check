package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
	k8sutil "github.com/vuldin/redpanda-check/internal/k8s"
)

// NodeIsolation validates that Redpanda pods have nodeSelector, nodeAffinity,
// or tolerations configured for dedicated-node scheduling.
func NodeIsolation(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "node_isolation",
		Description: "Dedicated node scheduling configured",
		Level:       checker.LevelRecommended,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "Node isolation check only applicable to Kubernetes deployments"
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

	pod := pods[0]
	spec := pod.Spec

	hasNodeSelector := len(spec.NodeSelector) > 0
	hasNodeAffinity := spec.Affinity != nil && spec.Affinity.NodeAffinity != nil
	// Filter out default tolerations that K8s adds automatically.
	hasTolerations := false
	for _, t := range spec.Tolerations {
		if t.Key != "node.kubernetes.io/not-ready" && t.Key != "node.kubernetes.io/unreachable" {
			hasTolerations = true
			break
		}
	}

	if hasNodeSelector || hasNodeAffinity || hasTolerations {
		var methods []string
		if hasNodeSelector {
			methods = append(methods, "nodeSelector")
		}
		if hasNodeAffinity {
			methods = append(methods, "nodeAffinity")
		}
		if hasTolerations {
			methods = append(methods, "tolerations")
		}
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("Dedicated node scheduling via: %v", methods)
	} else {
		r.Status = checker.StatusWarn
		r.Details = "No nodeSelector, nodeAffinity, or custom tolerations configured; Redpanda may share nodes with other workloads"
	}
	pc.AddResult(r)
}
