package k8s

import (
	"context"
	"fmt"
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClient creates a Kubernetes clientset. It first attempts in-cluster
// config, then falls back to the provided kubeconfig path (or the default
// loader if kubeconfigPath is empty).
func NewClient(kubeconfigPath string) (kubernetes.Interface, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		loadRules := clientcmd.NewDefaultClientConfigLoadingRules()
		if kubeconfigPath != "" {
			loadRules.ExplicitPath = kubeconfigPath
		}
		cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadRules,
			&clientcmd.ConfigOverrides{},
		).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("unable to load kubeconfig: %v", err)
		}
	}
	return kubernetes.NewForConfig(cfg)
}

// RedpandaPods returns the Redpanda pods in the given namespace by finding the
// headless service and listing pods that match its selector.
func RedpandaPods(ctx context.Context, client kubernetes.Interface, namespace string) ([]corev1.Pod, error) {
	svc, err := headlessService(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	podList, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String(),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list pods: %v", err)
	}
	return podList.Items, nil
}

// DiscoverAdminAddresses discovers Redpanda admin API addresses from the
// headless service in the given namespace. It returns addresses in the form
// "host:port".
func DiscoverAdminAddresses(ctx context.Context, client kubernetes.Interface, namespace string) ([]string, error) {
	svc, err := headlessService(ctx, client, namespace)
	if err != nil {
		return nil, err
	}

	podList, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(svc.Spec.Selector).String(),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list pods: %v", err)
	}

	domain := clusterDomain()
	var addrs []string
	for _, p := range podList.Items {
		for _, c := range p.Spec.Containers {
			for _, port := range c.Ports {
				if port.Name == "admin" {
					fqdn := fmt.Sprintf("%s.%s.%s.svc.%s",
						p.Spec.Hostname, svc.Name, p.Namespace, domain)
					addrs = append(addrs, fmt.Sprintf("%s:%d", fqdn, port.ContainerPort))
					break
				}
			}
		}
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no admin ports found in pods for namespace %s", namespace)
	}
	return addrs, nil
}

func headlessService(ctx context.Context, client kubernetes.Interface, namespace string) (*corev1.Service, error) {
	services, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to list services: %v", err)
	}
	for i := range services.Items {
		if services.Items[i].Spec.ClusterIP == corev1.ClusterIPNone {
			return &services.Items[i], nil
		}
	}
	return nil, fmt.Errorf("no headless service found in namespace %s", namespace)
}

func clusterDomain() string {
	cname, err := net.LookupCNAME("kubernetes.default.svc")
	if err != nil {
		return "cluster.local."
	}
	return strings.TrimPrefix(cname, "kubernetes.default.svc.")
}
