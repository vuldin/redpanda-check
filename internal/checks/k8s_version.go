package checks

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
	k8sutil "github.com/vuldin/redpanda-check/internal/k8s"
)

// minSupportedK8sMinor is the minimum Kubernetes minor version considered
// supported. K8s supports N-2 minor versions; update this as new versions
// release. As of 2026-04, latest is 1.32, so 1.29 is conservative.
const minSupportedK8sMinor = 29

// KubernetesVersion validates that the Kubernetes nodes running Redpanda
// are on a recent, supported version.
func KubernetesVersion(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "kubernetes_version",
		Description: "Kubernetes version is recent and supported",
		Level:       checker.LevelRecommended,
	}

	if !pc.IsK8s() {
		r.Status = checker.StatusSkip
		r.Details = "Kubernetes version check only applicable to Kubernetes deployments"
		pc.AddResult(r)
		return
	}

	nodes, err := k8sutil.DetectCloudNodes(ctx, pc.K8sClient, pc.Namespace)
	if err != nil {
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("Unable to detect node info: %v", err)
		pc.AddResult(r)
		return
	}

	if len(nodes) == 0 {
		r.Status = checker.StatusSkip
		r.Details = "No Redpanda nodes found"
		pc.AddResult(r)
		return
	}

	var outdated, current []string
	versions := make(map[string][]string)

	for _, node := range nodes {
		ver := node.KubeletVersion
		if ver == "" {
			continue
		}
		versions[ver] = append(versions[ver], node.NodeName)

		minor, err := parseK8sMinor(ver)
		if err != nil {
			continue
		}
		if minor < minSupportedK8sMinor {
			outdated = append(outdated, fmt.Sprintf("%s: %s", node.NodeName, ver))
		} else {
			current = append(current, fmt.Sprintf("%s: %s", node.NodeName, ver))
		}
	}

	if len(outdated) > 0 {
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("Outdated Kubernetes version detected (minimum supported: 1.%d):\n%s",
			minSupportedK8sMinor, strings.Join(outdated, "\n"))
		if len(current) > 0 {
			r.Details += "\nCurrent:\n" + strings.Join(current, "\n")
		}
	} else if len(versions) > 1 {
		var parts []string
		for ver, nodeNames := range versions {
			parts = append(parts, fmt.Sprintf("%s (%s)", ver, strings.Join(nodeNames, ", ")))
		}
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("Inconsistent Kubernetes versions across nodes: %s", strings.Join(parts, "; "))
	} else if len(current) > 0 {
		ver := nodes[0].KubeletVersion
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("All %d node(s) running Kubernetes %s", len(current), ver)
	} else {
		r.Status = checker.StatusSkip
		r.Details = "Unable to determine Kubernetes version"
	}
	pc.AddResult(r)
}

// parseK8sMinor extracts the minor version number from a kubelet version string
// like "v1.30.2" or "v1.29.1-eks-1234".
func parseK8sMinor(version string) (int, error) {
	v := strings.TrimPrefix(version, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid version format: %s", version)
	}
	return strconv.Atoi(parts[1])
}
