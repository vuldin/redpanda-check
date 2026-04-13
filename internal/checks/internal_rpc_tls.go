package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// InternalRPCTLS validates that TLS is enabled on the internal RPC server
// across all brokers.
func InternalRPCTLS(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "internal_rpc_tls",
		Description: "TLS enabled on internal RPC",
		Level:       checker.LevelCritical,
	}

	brokers, err := pc.BrokerList(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to list brokers: %v", err)
		pc.AddResult(r)
		return
	}

	configs, unreachable, err := pc.PerBrokerNodeConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get node config: %v", err)
		pc.AddResult(r)
		return
	}

	var missing []string
	for nodeID, nc := range configs {
		if !hasRPCTLS(nc) {
			missing = append(missing, nodeID)
		}
	}

	checked := len(configs)
	if len(missing) > 0 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Internal RPC TLS not enabled on brokers: %s", strings.Join(missing, ", "))
	} else {
		detail := fmt.Sprintf("Internal RPC TLS enabled on %d/%d brokers checked", checked, len(brokers))
		if unreachable > 0 {
			detail += fmt.Sprintf(" (%d brokers checked via single connection)", unreachable)
		}
		r.Status = checker.StatusPass
		r.Details = detail
	}
	pc.AddResult(r)
}

func hasRPCTLS(nodeConfig map[string]any) bool {
	val, ok := nodeConfig["rpc_server_tls"]
	if !ok {
		return false
	}
	m, ok := val.(map[string]any)
	if !ok {
		return false
	}
	enabled, ok := m["enabled"].(bool)
	return ok && enabled
}
