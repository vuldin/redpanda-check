package checker

import (
	"context"
	"fmt"
	"sync"

	"github.com/redpanda-data/common-go/rpadmin"
	"k8s.io/client-go/kubernetes"
)

// ProductionChecker orchestrates production readiness checks. It holds shared
// clients that individual checks use, avoiding redundant connection setup.
// Cached accessors (ClusterConfig, BrokerList) ensure expensive API calls are
// made at most once per run.
type ProductionChecker struct {
	AdminClient *rpadmin.AdminAPI
	K8sClient   kubernetes.Interface
	Namespace   string
	Results     []CheckResult

	configOnce    sync.Once
	cachedConfig  map[string]any
	configErr     error

	brokersOnce    sync.Once
	cachedBrokers  []rpadmin.Broker
	brokersErr     error
}

// IsK8s returns true when a Kubernetes client is available.
func (pc *ProductionChecker) IsK8s() bool {
	return pc.K8sClient != nil
}

// AddResult appends a check result.
func (pc *ProductionChecker) AddResult(r CheckResult) {
	pc.Results = append(pc.Results, r)
}

// ClusterConfig returns the cluster config, fetching it once and caching the
// result. Multiple checks that need Config() share a single API call.
func (pc *ProductionChecker) ClusterConfig(ctx context.Context) (map[string]any, error) {
	pc.configOnce.Do(func() {
		pc.cachedConfig, pc.configErr = pc.AdminClient.Config(ctx, true)
	})
	return pc.cachedConfig, pc.configErr
}

// BrokerList returns the broker list, fetching it once and caching the result.
func (pc *ProductionChecker) BrokerList(ctx context.Context) ([]rpadmin.Broker, error) {
	pc.brokersOnce.Do(func() {
		pc.cachedBrokers, pc.brokersErr = pc.AdminClient.Brokers(ctx)
	})
	return pc.cachedBrokers, pc.brokersErr
}

// PerBrokerNodeConfig queries RawNodeConfig on each broker individually via
// ForBroker. If ForBroker fails (e.g. port-forward scenario), it falls back to
// querying the connected broker only. Returns a map of broker ID string to its
// raw node config, plus the count of brokers that could not be reached.
func (pc *ProductionChecker) PerBrokerNodeConfig(ctx context.Context) (map[string]map[string]any, int, error) {
	brokers, err := pc.BrokerList(ctx)
	if err != nil {
		return nil, 0, err
	}

	configs := make(map[string]map[string]any)
	unreachable := 0

	for _, b := range brokers {
		bc, err := pc.AdminClient.ForBroker(ctx, b.NodeID)
		if err != nil {
			unreachable++
			continue
		}
		nc, err := bc.RawNodeConfig(ctx)
		bc.Close()
		if err != nil {
			unreachable++
			continue
		}
		configs[fmt.Sprintf("%d", b.NodeID)] = nc
	}

	// Fallback: if no individual brokers were reachable, query the connected one.
	if len(configs) == 0 {
		nc, err := pc.AdminClient.RawNodeConfig(ctx)
		if err != nil {
			return nil, len(brokers), err
		}
		configs["connected"] = nc
		unreachable = len(brokers) - 1
	}

	return configs, unreachable, nil
}

// Check is the function signature every individual check implements.
type Check func(ctx context.Context, pc *ProductionChecker)
