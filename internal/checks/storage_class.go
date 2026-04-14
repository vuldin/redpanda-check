package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
	k8sutil "github.com/vuldin/redpanda-check/internal/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// recommendedProvisioners maps cloud providers to their recommended CSI provisioners.
var recommendedProvisioners = map[k8sutil.CloudProvider][]string{
	k8sutil.CloudAWS:   {"ebs.csi.aws.com"},
	k8sutil.CloudGCP:   {"pd.csi.storage.gke.io"},
	k8sutil.CloudAzure: {"disk.csi.azure.com"},
}

// deprecatedProvisioners maps old in-tree provisioner names.
var deprecatedProvisioners = map[string]string{
	"kubernetes.io/aws-ebs":    "ebs.csi.aws.com",
	"kubernetes.io/gce-pd":     "pd.csi.storage.gke.io",
	"kubernetes.io/azure-disk": "disk.csi.azure.com",
}

// recommendedParams maps cloud providers to their recommended StorageClass parameter values.
var recommendedParams = map[k8sutil.CloudProvider]map[string][]string{
	k8sutil.CloudAWS: {
		"type": {"gp3", "io2", "io1"},
	},
	k8sutil.CloudGCP: {
		"type": {"pd-ssd"},
	},
	k8sutil.CloudAzure: {
		"skuname": {"Premium_LRS", "UltraSSD_LRS"},
	},
}

// StorageClassValidation validates that Redpanda PVCs use recommended
// StorageClass provisioners and parameters for the detected cloud provider.
func StorageClassValidation(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "storage_class_validation",
		Description: "StorageClass uses recommended provisioner and parameters",
		Level:       checker.LevelRecommended,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "StorageClass check only applicable to Kubernetes deployments"
		pc.AddResult(r)
		return
	}

	// Detect cloud provider.
	nodes, err := k8sutil.DetectCloudNodes(ctx, pc.K8sClient, pc.Namespace)
	if err != nil {
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("Unable to detect cloud environment: %v", err)
		pc.AddResult(r)
		return
	}

	cloud := k8sutil.CloudUnknown
	if len(nodes) > 0 {
		cloud = nodes[0].CloudProvider
	}

	// List PVCs in namespace.
	pvcs, err := pc.K8sClient.CoreV1().PersistentVolumeClaims(pc.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		r.Status = checker.StatusWarn
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

	var warnings, good []string

	for _, pvc := range pvcs.Items {
		scName := ""
		if pvc.Spec.StorageClassName != nil {
			scName = *pvc.Spec.StorageClassName
		}
		if scName == "" {
			warnings = append(warnings, fmt.Sprintf("%s: no StorageClass specified", pvc.Name))
			continue
		}

		sc, err := pc.K8sClient.StorageV1().StorageClasses().Get(ctx, scName, metav1.GetOptions{})
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: unable to fetch StorageClass %q", pvc.Name, scName))
			continue
		}

		provisioner := sc.Provisioner

		// Check for deprecated provisioner.
		if replacement, ok := deprecatedProvisioners[provisioner]; ok {
			warnings = append(warnings, fmt.Sprintf("%s: using deprecated provisioner %q (recommend: %s)", pvc.Name, provisioner, replacement))
			continue
		}

		// Check provisioner matches cloud-recommended (only if cloud is known).
		if cloud != k8sutil.CloudUnknown {
			recommended := recommendedProvisioners[cloud]
			provisionerOk := false
			for _, rp := range recommended {
				if provisioner == rp {
					provisionerOk = true
					break
				}
			}
			// Also accept local-volume provisioners.
			if isLocalProvisioner(provisioner) {
				provisionerOk = true
			}
			if !provisionerOk {
				warnings = append(warnings, fmt.Sprintf("%s: provisioner %q not in recommended list for %s", pvc.Name, provisioner, cloud))
				continue
			}
		}

		// Check parameters (only for known cloud + network-attached provisioners).
		if cloud != k8sutil.CloudUnknown && !isLocalProvisioner(provisioner) {
			if paramWarning := checkParams(cloud, sc.Parameters, pvc.Name); paramWarning != "" {
				warnings = append(warnings, paramWarning)
				continue
			}
		}

		detail := fmt.Sprintf("%s: %s (%s)", pvc.Name, scName, provisioner)
		if cloud != k8sutil.CloudUnknown {
			detail += fmt.Sprintf(" [%s]", cloud)
		}
		good = append(good, detail)
	}

	if len(warnings) > 0 {
		r.Status = checker.StatusWarn
		r.Details = "StorageClass issues:\n" + strings.Join(warnings, "\n")
		if len(good) > 0 {
			r.Details += "\nPassing:\n" + strings.Join(good, "\n")
		}
	} else {
		r.Status = checker.StatusPass
		r.Details = "StorageClass configuration is appropriate:\n" + strings.Join(good, "\n")
	}
	pc.AddResult(r)
}

func checkParams(cloud k8sutil.CloudProvider, params map[string]string, pvcName string) string {
	rp, ok := recommendedParams[cloud]
	if !ok {
		return ""
	}
	for key, allowed := range rp {
		val, exists := params[key]
		if !exists {
			// Param key not set; some clouds use defaults.
			continue
		}
		found := false
		for _, a := range allowed {
			if strings.EqualFold(val, a) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Sprintf("%s: StorageClass parameter %s=%q not recommended for %s (recommend: %s)",
				pvcName, key, val, cloud, strings.Join(allowed, ", "))
		}
	}
	return ""
}

func isLocalProvisioner(provisioner string) bool {
	lower := strings.ToLower(provisioner)
	return strings.Contains(lower, "local") ||
		strings.Contains(lower, "lvm") ||
		strings.Contains(lower, "topolvm")
}
