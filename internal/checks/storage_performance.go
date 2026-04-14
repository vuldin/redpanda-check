package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
	k8sutil "github.com/vuldin/redpanda-check/internal/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// networkAttachedProvisioners are CSI drivers that use network-attached storage.
var networkAttachedProvisioners = map[string]bool{
	"ebs.csi.aws.com":          true,
	"pd.csi.storage.gke.io":    true,
	"disk.csi.azure.com":       true,
	"kubernetes.io/aws-ebs":    true,
	"kubernetes.io/gce-pd":     true,
	"kubernetes.io/azure-disk": true,
}

// StoragePerformance validates that Redpanda uses local NVMe-backed storage
// rather than network-attached volumes. Redpanda's production requirements
// specify local NVMe drives for optimal performance.
func StoragePerformance(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "storage_performance",
		Description: "Local NVMe storage (not network-attached)",
		Level:       checker.LevelCritical,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "Storage performance check only applicable to Kubernetes deployments"
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
		r.Status = checker.StatusSkip
		r.Details = "No PVCs found"
		pc.AddResult(r)
		return
	}

	// Gather instance type info for context.
	var instanceInfo string
	nodes, err := k8sutil.DetectCloudNodes(ctx, pc.K8sClient, pc.Namespace)
	if err == nil && len(nodes) > 0 {
		types := make(map[string]bool)
		for _, n := range nodes {
			if n.InstanceType != "" {
				types[n.InstanceType] = true
			}
		}
		if len(types) > 0 {
			var typeList []string
			for t := range types {
				typeList = append(typeList, t)
			}
			instanceInfo = fmt.Sprintf(" (instance types: %s)", strings.Join(typeList, ", "))
		}
	}

	var localPVCs, networkPVCs, unknownPVCs []string

	for _, pvc := range pvcs.Items {
		scName := ""
		if pvc.Spec.StorageClassName != nil {
			scName = *pvc.Spec.StorageClassName
		}
		if scName == "" {
			unknownPVCs = append(unknownPVCs, pvc.Name)
			continue
		}

		sc, err := pc.K8sClient.StorageV1().StorageClasses().Get(ctx, scName, metav1.GetOptions{})
		if err != nil {
			unknownPVCs = append(unknownPVCs, pvc.Name)
			continue
		}

		provisioner := sc.Provisioner

		if isLocalProvisioner(provisioner) {
			localPVCs = append(localPVCs, fmt.Sprintf("%s (%s)", pvc.Name, provisioner))
		} else if networkAttachedProvisioners[provisioner] {
			networkPVCs = append(networkPVCs, fmt.Sprintf("%s (%s)", pvc.Name, provisioner))
		} else if strings.Contains(strings.ToLower(provisioner), "hostpath") {
			networkPVCs = append(networkPVCs, fmt.Sprintf("%s (%s — hostPath)", pvc.Name, provisioner))
		} else {
			unknownPVCs = append(unknownPVCs, fmt.Sprintf("%s (%s)", pvc.Name, provisioner))
		}
	}

	if len(networkPVCs) > 0 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Network-attached storage detected; Redpanda requires local NVMe for production:\n%s",
			strings.Join(networkPVCs, "\n"))
		if instanceInfo != "" {
			r.Details += "\n" + instanceInfo
		}
	} else if len(localPVCs) > 0 {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("Local storage provisioner detected:\n%s", strings.Join(localPVCs, "\n"))
		if instanceInfo != "" {
			r.Details += "\n" + instanceInfo
		}
	} else if len(unknownPVCs) > 0 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to determine if storage is local NVMe:\n%s",
			strings.Join(unknownPVCs, "\n"))
	} else {
		r.Status = checker.StatusSkip
		r.Details = "No PVCs to evaluate"
	}
	pc.AddResult(r)
}
