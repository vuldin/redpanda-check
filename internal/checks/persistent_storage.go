package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PersistentStorage validates persistent storage configuration (no hostPath).
func PersistentStorage(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "persistent_storage",
		Description: "Persistent storage configuration (no hostPath)",
		Level:       checker.LevelCritical,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "Storage check only applicable to Kubernetes deployments"
		pc.AddResult(r)
		return
	}

	pvcs, err := pc.K8sClient.CoreV1().PersistentVolumeClaims(pc.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to list PVCs: %v", err)
		pc.AddResult(r)
		return
	}

	if len(pvcs.Items) == 0 {
		r.Status = checker.StatusFail
		r.Details = "No PVCs found for Redpanda deployment"
		pc.AddResult(r)
		return
	}

	hostpathFound := false
	var details []string

	for _, pvc := range pvcs.Items {
		scName := ""
		if pvc.Spec.StorageClassName != nil {
			scName = *pvc.Spec.StorageClassName
		}
		if scName != "" {
			sc, err := pc.K8sClient.StorageV1().StorageClasses().Get(ctx, scName, metav1.GetOptions{})
			if err == nil {
				if strings.Contains(strings.ToLower(sc.Provisioner), "hostpath") {
					hostpathFound = true
				}
				details = append(details, fmt.Sprintf("%s: %s (%s)", pvc.Name, scName, sc.Provisioner))
			} else {
				details = append(details, fmt.Sprintf("%s: %s", pvc.Name, scName))
			}
		} else {
			details = append(details, fmt.Sprintf("%s: no storage class", pvc.Name))
		}
	}

	if hostpathFound {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("HostPath storage detected (not recommended for production):\n%s",
			strings.Join(details, "\n"))
	} else {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("Using persistent storage:\n%s", strings.Join(details, "\n"))
	}
	pc.AddResult(r)
}
