package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// AdvertisedAddresses validates that advertised Kafka API and RPC addresses
// are properly configured with routable addresses (not 0.0.0.0 or empty).
func AdvertisedAddresses(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "advertised_addresses",
		Description: "Advertised addresses configured",
		Level:       checker.LevelCritical,
	}

	configs, unreachable, err := pc.PerBrokerNodeConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get node config: %v", err)
		pc.AddResult(r)
		return
	}

	var issues, good []string

	for brokerID, nc := range configs {
		brokerIssues := checkAdvertisedKafkaAPI(nc, brokerID)
		brokerIssues = append(brokerIssues, checkAdvertisedRPCAPI(nc, brokerID)...)

		if len(brokerIssues) > 0 {
			issues = append(issues, brokerIssues...)
		} else {
			good = append(good, brokerID)
		}
	}

	if len(issues) > 0 {
		r.Status = checker.StatusFail
		r.Details = "Advertised address issues:\n" + strings.Join(issues, "\n")
	} else {
		detail := fmt.Sprintf("Advertised addresses configured on %d broker(s)", len(good))
		if unreachable > 0 {
			detail += fmt.Sprintf(" (%d brokers checked via single connection)", unreachable)
		}
		r.Status = checker.StatusPass
		r.Details = detail
	}
	pc.AddResult(r)
}

func checkAdvertisedKafkaAPI(nc map[string]any, brokerID string) []string {
	var issues []string

	val, ok := nc["advertised_kafka_api"]
	if !ok {
		issues = append(issues, fmt.Sprintf("broker %s: advertised_kafka_api not configured", brokerID))
		return issues
	}

	entries, ok := val.([]any)
	if !ok || len(entries) == 0 {
		issues = append(issues, fmt.Sprintf("broker %s: advertised_kafka_api is empty", brokerID))
		return issues
	}

	for _, entry := range entries {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		addr, _ := m["address"].(string)
		if addr == "" || addr == "0.0.0.0" {
			issues = append(issues, fmt.Sprintf("broker %s: advertised_kafka_api has non-routable address %q", brokerID, addr))
		}
	}

	return issues
}

func checkAdvertisedRPCAPI(nc map[string]any, brokerID string) []string {
	var issues []string

	val, ok := nc["advertised_rpc_api"]
	if !ok {
		issues = append(issues, fmt.Sprintf("broker %s: advertised_rpc_api not configured", brokerID))
		return issues
	}

	m, ok := val.(map[string]any)
	if !ok {
		issues = append(issues, fmt.Sprintf("broker %s: advertised_rpc_api has unexpected format", brokerID))
		return issues
	}

	addr, _ := m["address"].(string)
	if addr == "" || addr == "0.0.0.0" {
		issues = append(issues, fmt.Sprintf("broker %s: advertised_rpc_api has non-routable address %q", brokerID, addr))
	}

	return issues
}
