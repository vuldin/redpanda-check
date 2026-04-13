package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodDisruptionBudget validates a PDB is configured for the namespace.
func PodDisruptionBudget(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "pod_disruption_budget",
		Description: "Pod Disruption Budget configured",
		Level:       checker.LevelCritical,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "PDB check only applicable to Kubernetes deployments"
		pc.AddResult(r)
		return
	}

	pdbs, err := pc.K8sClient.PolicyV1().PodDisruptionBudgets(pc.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to list PDBs: %v", err)
		pc.AddResult(r)
		return
	}

	if len(pdbs.Items) == 0 {
		r.Status = checker.StatusFail
		r.Details = "No Pod Disruption Budget found"
		pc.AddResult(r)
		return
	}

	var details []string
	for _, pdb := range pdbs.Items {
		switch {
		case pdb.Spec.MinAvailable != nil:
			details = append(details, fmt.Sprintf("%s (minAvailable: %s)", pdb.Name, pdb.Spec.MinAvailable.String()))
		case pdb.Spec.MaxUnavailable != nil:
			details = append(details, fmt.Sprintf("%s (maxUnavailable: %s)", pdb.Name, pdb.Spec.MaxUnavailable.String()))
		default:
			details = append(details, pdb.Name)
		}
	}

	r.Status = checker.StatusPass
	r.Details = fmt.Sprintf("PDBs found: %s", strings.Join(details, ", "))
	pc.AddResult(r)
}
