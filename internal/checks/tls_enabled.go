package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// TLSEnabled validates that TLS is configured on Kafka API and admin API
// listeners by inspecting the node configuration of every broker.
func TLSEnabled(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "tls_enabled",
		Description: "TLS enabled on Kafka and Admin API listeners",
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

	type listenerCheck struct {
		configKey string
		label     string
	}
	listeners := []listenerCheck{
		{"admin_api_tls", "Admin API"},
		{"kafka_api_tls", "Kafka API"},
	}

	missingByListener := make(map[string][]string)
	enabledByListener := make(map[string][]string)

	for nodeID, nc := range configs {
		for _, lc := range listeners {
			if hasTLSListener(nc, lc.configKey) {
				enabledByListener[lc.label] = append(enabledByListener[lc.label], nodeID)
			} else {
				missingByListener[lc.label] = append(missingByListener[lc.label], nodeID)
			}
		}
	}

	var issues []string
	for listener, brokerIDs := range missingByListener {
		issues = append(issues, fmt.Sprintf("%s missing on brokers: %s", listener, strings.Join(brokerIDs, ", ")))
	}

	if len(issues) > 0 {
		r.Status = checker.StatusFail
		r.Details = "TLS issues: " + strings.Join(issues, "; ")
	} else {
		var parts []string
		for listener := range enabledByListener {
			parts = append(parts, listener)
		}
		checked := len(configs)
		detail := fmt.Sprintf("TLS configured on %d/%d brokers checked: %s",
			checked, len(brokers), strings.Join(parts, ", "))
		if unreachable > 0 {
			detail += fmt.Sprintf(" (%d brokers checked via single connection)", unreachable)
		}
		r.Status = checker.StatusPass
		r.Details = detail
	}
	pc.AddResult(r)
}

// hasTLSListener checks whether a TLS config key contains at least one
// listener with "enabled": true.
func hasTLSListener(nodeConfig map[string]any, key string) bool {
	val, ok := nodeConfig[key]
	if !ok {
		return false
	}
	entries, ok := val.([]any)
	if !ok || len(entries) == 0 {
		return false
	}
	for _, entry := range entries {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if enabled, ok := m["enabled"].(bool); ok && enabled {
			return true
		}
	}
	return false
}
