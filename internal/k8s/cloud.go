package k8s

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CloudProvider represents a detected cloud environment.
type CloudProvider string

const (
	CloudAWS     CloudProvider = "aws"
	CloudGCP     CloudProvider = "gcp"
	CloudAzure   CloudProvider = "azure"
	CloudUnknown CloudProvider = "unknown"
)

// NodeInfo holds cloud-relevant metadata for a Kubernetes node running Redpanda.
type NodeInfo struct {
	NodeName       string
	CloudProvider  CloudProvider
	InstanceType   string
	KubeletVersion string
}

// DetectCloudNodes returns cloud provider and instance metadata for each node
// running a Redpanda pod. It looks up pod.Spec.NodeName for each Redpanda pod,
// then queries the node API for ProviderID, instance type label, and kubelet version.
func DetectCloudNodes(ctx context.Context, client kubernetes.Interface, namespace string) ([]NodeInfo, error) {
	pods, err := RedpandaPods(ctx, client, namespace)
	if err != nil {
		return nil, fmt.Errorf("unable to list Redpanda pods: %v", err)
	}

	// Collect unique node names.
	nodeNames := make(map[string]bool)
	for _, pod := range pods {
		if pod.Spec.NodeName != "" {
			nodeNames[pod.Spec.NodeName] = true
		}
	}

	var nodes []NodeInfo
	for name := range nodeNames {
		node, err := client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			continue
		}

		info := NodeInfo{
			NodeName:       name,
			CloudProvider:  parseProviderID(node.Spec.ProviderID),
			InstanceType:   node.Labels["node.kubernetes.io/instance-type"],
			KubeletVersion: node.Status.NodeInfo.KubeletVersion,
		}

		// Fall back to beta label if standard label is missing.
		if info.InstanceType == "" {
			info.InstanceType = node.Labels["beta.kubernetes.io/instance-type"]
		}

		nodes = append(nodes, info)
	}

	return nodes, nil
}

// parseProviderID extracts the cloud provider from a node's ProviderID.
// Format is typically "provider://region/instance-id" or "provider:///instance-id".
func parseProviderID(providerID string) CloudProvider {
	if providerID == "" {
		return CloudUnknown
	}
	lower := strings.ToLower(providerID)
	switch {
	case strings.HasPrefix(lower, "aws"):
		return CloudAWS
	case strings.HasPrefix(lower, "gce"):
		return CloudGCP
	case strings.HasPrefix(lower, "azure"):
		return CloudAzure
	default:
		return CloudUnknown
	}
}
