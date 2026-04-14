package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkPolicies validates that at least one NetworkPolicy exists in the
// Redpanda namespace to restrict pod-to-pod traffic.
func NetworkPolicies(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "network_policies",
		Description: "NetworkPolicies configured",
		Level:       checker.LevelRecommended,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "NetworkPolicy check only applicable to Kubernetes deployments"
		pc.AddResult(r)
		return
	}

	policies, err := pc.K8sClient.NetworkingV1().NetworkPolicies(pc.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("Unable to list NetworkPolicies: %v", err)
		pc.AddResult(r)
		return
	}

	if len(policies.Items) == 0 {
		r.Status = checker.StatusWarn
		r.Details = "No NetworkPolicies configured; pod traffic is unrestricted"
	} else {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("%d NetworkPolicy(ies) configured in namespace", len(policies.Items))
	}
	pc.AddResult(r)
}
