package checks_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/redpanda-data/common-go/rpadmin"
	"github.com/vuldin/redpanda-check/internal/checker"
	"github.com/vuldin/redpanda-check/internal/checks"
)

// newTestChecker creates a ProductionChecker backed by a fake admin API server
// that responds to the provided handlers. The caller provides a map of URL
// path → handler func. Unmatched paths return 404.
func newTestChecker(t *testing.T, handlers map[string]http.HandlerFunc) *checker.ProductionChecker {
	t.Helper()
	mux := http.NewServeMux()
	for path, handler := range handlers {
		mux.HandleFunc(path, handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client, err := rpadmin.NewAdminAPI([]string{srv.URL}, &rpadmin.NopAuth{}, nil)
	if err != nil {
		t.Fatalf("unable to create admin client: %v", err)
	}
	t.Cleanup(client.Close)

	return &checker.ProductionChecker{
		AdminClient: client,
		Namespace:   "redpanda",
	}
}

func jsonHandler(t *testing.T, v any) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
}

// --- ClusterHealth ---

func TestClusterHealth_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster/health_overview": jsonHandler(t, rpadmin.ClusterHealthOverview{
			IsHealthy: true,
			AllNodes:  []int{0, 1, 2},
		}),
	})
	checks.ClusterHealth(context.Background(), pc)

	if len(pc.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(pc.Results))
	}
	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestClusterHealth_Fail(t *testing.T) {
	lc := 5
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster/health_overview": jsonHandler(t, rpadmin.ClusterHealthOverview{
			IsHealthy:        false,
			UnhealthyReasons: []string{"leaderless partitions"},
			NodesDown:        []int{2},
			LeaderlessCount:  &lc,
		}),
	})
	checks.ClusterHealth(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusFail {
		t.Errorf("expected FAIL, got %s", pc.Results[0].Status)
	}
}

// --- BrokerCount ---

func TestBrokerCount_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/brokers": jsonHandler(t, []rpadmin.Broker{
			{NodeID: 0}, {NodeID: 1}, {NodeID: 2},
		}),
	})
	checks.BrokerCount(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestBrokerCount_Warn(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/brokers": jsonHandler(t, []rpadmin.Broker{
			{NodeID: 0},
		}),
	})
	checks.BrokerCount(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusWarn {
		t.Errorf("expected WARN, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- BrokerMembership ---

func TestBrokerMembership_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/brokers": jsonHandler(t, []rpadmin.Broker{
			{NodeID: 0, MembershipStatus: "active"},
			{NodeID: 1, MembershipStatus: "active"},
		}),
	})
	checks.BrokerMembership(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestBrokerMembership_Fail(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/brokers": jsonHandler(t, []rpadmin.Broker{
			{NodeID: 0, MembershipStatus: "active"},
			{NodeID: 1, MembershipStatus: "draining"},
		}),
	})
	checks.BrokerMembership(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusFail {
		t.Errorf("expected FAIL, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- MaintenanceMode ---

func TestMaintenanceMode_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/brokers": jsonHandler(t, []rpadmin.Broker{
			{NodeID: 0, Maintenance: nil},
			{NodeID: 1, Maintenance: &rpadmin.MaintenanceStatus{Draining: false}},
		}),
	})
	checks.MaintenanceMode(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestMaintenanceMode_Fail(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/brokers": jsonHandler(t, []rpadmin.Broker{
			{NodeID: 0, Maintenance: &rpadmin.MaintenanceStatus{Draining: true}},
		}),
	})
	checks.MaintenanceMode(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusFail {
		t.Errorf("expected FAIL, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- VersionConsistency ---

func TestVersionConsistency_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/brokers": jsonHandler(t, []rpadmin.Broker{
			{NodeID: 0, Version: "v24.1.1"},
			{NodeID: 1, Version: "v24.1.1"},
		}),
	})
	checks.VersionConsistency(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestVersionConsistency_Fail(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/brokers": jsonHandler(t, []rpadmin.Broker{
			{NodeID: 0, Version: "v24.1.1"},
			{NodeID: 1, Version: "v24.2.0"},
		}),
	})
	checks.VersionConsistency(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusFail {
		t.Errorf("expected FAIL, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- DeveloperMode ---

func TestDeveloperMode_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"developer_mode": false,
		}),
	})
	checks.DeveloperMode(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestDeveloperMode_Fail(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"developer_mode": true,
		}),
	})
	checks.DeveloperMode(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusFail {
		t.Errorf("expected FAIL, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- Overprovisioned ---

func TestOverprovisioned_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"overprovisioned": false,
		}),
	})
	checks.Overprovisioned(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- ReplicationFactor ---

func TestReplicationFactor_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"default_topic_replications": float64(3),
		}),
	})
	checks.ReplicationFactor(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestReplicationFactor_Fail(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"default_topic_replications": float64(1),
		}),
	})
	checks.ReplicationFactor(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusFail {
		t.Errorf("expected FAIL, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- Authorization ---

func TestAuthorization_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"kafka_enable_authorization": true,
			"enable_sasl":               true,
		}),
	})
	checks.Authorization(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestAuthorization_Fail(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"kafka_enable_authorization": false,
			"enable_sasl":               false,
		}),
	})
	checks.Authorization(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusFail {
		t.Errorf("expected FAIL, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- License ---

func TestLicense_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/features/license": jsonHandler(t, rpadmin.License{
			Loaded: true,
			Properties: rpadmin.LicenseProperties{
				Type:         "enterprise",
				Organization: "test-org",
				Expires:      4102444800, // 2100-01-01
			},
		}),
	})
	checks.License(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestLicense_Warn_NotLoaded(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/features/license": jsonHandler(t, rpadmin.License{
			Loaded: false,
		}),
	})
	checks.License(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusWarn {
		t.Errorf("expected WARN, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- ExistingTopicsReplication ---

func TestExistingTopicsReplication_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster/partitions": jsonHandler(t, []rpadmin.ClusterPartition{
			{Ns: "kafka", Topic: "test-topic", PartitionID: 0, Replicas: []rpadmin.Replica{{NodeID: 0}, {NodeID: 1}, {NodeID: 2}}},
		}),
	})
	checks.ExistingTopicsReplication(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestExistingTopicsReplication_Fail(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster/partitions": jsonHandler(t, []rpadmin.ClusterPartition{
			{Ns: "kafka", Topic: "under-rep", PartitionID: 0, Replicas: []rpadmin.Replica{{NodeID: 0}}},
		}),
	})
	checks.ExistingTopicsReplication(context.Background(), pc)

	if pc.Results[0].Status != checker.StatusFail {
		t.Errorf("expected FAIL, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- DataBalancing ---

func TestDataBalancing_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"partition_autobalancing_mode": "continuous",
		}),
	})
	checks.DataBalancing(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestDataBalancing_Warn(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"partition_autobalancing_mode": "off",
		}),
	})
	checks.DataBalancing(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusWarn {
		t.Errorf("expected WARN, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- RackAwareness ---

func TestRackAwareness_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"enable_rack_awareness": true,
		}),
		"/v1/brokers": jsonHandler(t, []rpadmin.Broker{
			{NodeID: 0, Rack: "us-east-1a"},
			{NodeID: 1, Rack: "us-east-1b"},
		}),
	})
	checks.RackAwareness(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestRackAwareness_Warn_Disabled(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"enable_rack_awareness": false,
		}),
	})
	checks.RackAwareness(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusWarn {
		t.Errorf("expected WARN, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- MinTopicReplications ---

func TestMinTopicReplications_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"minimum_topic_replications": float64(3),
		}),
	})
	checks.MinTopicReplications(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestMinTopicReplications_Fail(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"minimum_topic_replications": float64(1),
		}),
	})
	checks.MinTopicReplications(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusFail {
		t.Errorf("expected FAIL, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- TieredStorage ---

func TestTieredStorage_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"cloud_storage_enabled": true,
			"cloud_storage_bucket":  "my-bucket",
			"cloud_storage_region":  "us-east-1",
		}),
	})
	checks.TieredStorage(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- AuditLogging ---

func TestAuditLogging_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"audit_enabled": true,
		}),
	})
	checks.AuditLogging(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- CoreBalancing ---

func TestCoreBalancing_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster_config": jsonHandler(t, map[string]any{
			"core_balancing_on_core_count_change": true,
		}),
	})
	checks.CoreBalancing(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- EnterpriseLicense ---

func TestEnterpriseLicense_Pass_NoFeatures(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/features/enterprise": jsonHandler(t, rpadmin.EnterpriseFeaturesResponse{
			LicenseStatus: "valid",
			Violation:     false,
			Features:      []rpadmin.EnterpriseFeature{},
		}),
	})
	checks.EnterpriseLicense(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestEnterpriseLicense_Fail_Violation(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/features/enterprise": jsonHandler(t, rpadmin.EnterpriseFeaturesResponse{
			LicenseStatus: "expired",
			Violation:     true,
			Features: []rpadmin.EnterpriseFeature{
				{Name: "audit_logging", Enabled: true},
			},
		}),
	})
	checks.EnterpriseLicense(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusFail {
		t.Errorf("expected FAIL, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- PartitionBalancerStatus ---

func TestPartitionBalancerStatus_Pass(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster/partition_balancer/status": jsonHandler(t, map[string]any{
			"status": "ready",
		}),
	})
	checks.PartitionBalancerStatus(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusPass {
		t.Errorf("expected PASS, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

func TestPartitionBalancerStatus_Stalled(t *testing.T) {
	pc := newTestChecker(t, map[string]http.HandlerFunc{
		"/v1/cluster/partition_balancer/status": jsonHandler(t, map[string]any{
			"status": "stalled",
		}),
	})
	checks.PartitionBalancerStatus(context.Background(), pc)
	if pc.Results[0].Status != checker.StatusWarn {
		t.Errorf("expected WARN, got %s: %s", pc.Results[0].Status, pc.Results[0].Details)
	}
}

// --- K8s checks skip when no K8s client ---

func TestK8sChecks_SkipWithoutClient(t *testing.T) {
	// These checks should all SKIP when K8sClient is nil.
	pc := &checker.ProductionChecker{
		Namespace: "redpanda",
	}

	k8sChecks := []struct {
		name  string
		check checker.Check
	}{
		{"PersistentStorage", checks.PersistentStorage},
		{"ResourceLimits", checks.ResourceLimits},
		{"PodDisruptionBudget", checks.PodDisruptionBudget},
		{"CPUMemoryRatio", checks.CPUMemoryRatio},
		{"NoFractionalCPU", checks.NoFractionalCPU},
		{"TopologySpread", checks.TopologySpread},
		{"NodeIsolation", checks.NodeIsolation},
	}

	for _, tc := range k8sChecks {
		t.Run(tc.name, func(t *testing.T) {
			before := len(pc.Results)
			tc.check(context.Background(), pc)
			if len(pc.Results) <= before {
				t.Fatal("expected a result to be added")
			}
			r := pc.Results[len(pc.Results)-1]
			if r.Status != checker.StatusSkip {
				t.Errorf("expected SKIP, got %s: %s", r.Status, r.Details)
			}
		})
	}
}
