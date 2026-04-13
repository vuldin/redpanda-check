package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
	k8sutil "github.com/vuldin/redpanda-check/internal/k8s"
)

// TopologySpread validates that Redpanda pods have topology spread constraints
// or pod anti-affinity configured to distribute brokers across failure domains.
func TopologySpread(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "topology_spread",
		Description: "Pod topology spread or anti-affinity configured",
		Level:       checker.LevelRecommended,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "Topology spread check only applicable to Kubernetes deployments"
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

	// Check the first pod's spec -- all pods in a StatefulSet share the same
	// template, so checking one is sufficient.
	pod := pods[0]
	spec := pod.Spec

	hasTopologySpread := len(spec.TopologySpreadConstraints) > 0
	hasAntiAffinity := spec.Affinity != nil &&
		spec.Affinity.PodAntiAffinity != nil &&
		(len(spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 ||
			len(spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0)

	if hasTopologySpread && hasAntiAffinity {
		r.Status = checker.StatusPass
		r.Details = "Both topologySpreadConstraints and podAntiAffinity configured"
	} else if hasTopologySpread {
		r.Status = checker.StatusPass
		r.Details = "topologySpreadConstraints configured"
	} else if hasAntiAffinity {
		r.Status = checker.StatusPass
		r.Details = "podAntiAffinity configured"
	} else {
		r.Status = checker.StatusWarn
		r.Details = "No topologySpreadConstraints or podAntiAffinity configured; brokers may be co-located"
	}
	pc.AddResult(r)
}
