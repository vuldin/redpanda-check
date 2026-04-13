package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// RackAwareness validates rack awareness is enabled and all brokers have
// rack IDs assigned.
func RackAwareness(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "rack_awareness",
		Description: "Rack awareness enabled with rack IDs assigned",
		Level:       checker.LevelRecommended,
	}

	config, err := pc.ClusterConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get cluster config: %v", err)
		pc.AddResult(r)
		return
	}

	if !boolFromConfig(config, "enable_rack_awareness") {
		r.Status = checker.StatusWarn
		r.Details = "Rack awareness is disabled (enable_rack_awareness: false)"
		pc.AddResult(r)
		return
	}

	brokers, err := pc.BrokerList(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Rack awareness enabled but unable to verify broker racks: %v", err)
		pc.AddResult(r)
		return
	}

	var missing []string
	racks := make(map[string]int)
	for _, b := range brokers {
		if b.Rack == "" {
			missing = append(missing, fmt.Sprintf("%d", b.NodeID))
		} else {
			racks[b.Rack]++
		}
	}

	if len(missing) > 0 {
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("Rack awareness enabled but brokers missing rack ID: %s", strings.Join(missing, ", "))
	} else {
		var rackInfo []string
		for rack, count := range racks {
			rackInfo = append(rackInfo, fmt.Sprintf("%s: %d brokers", rack, count))
		}
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("Rack awareness enabled (%s)", strings.Join(rackInfo, ", "))
	}
	pc.AddResult(r)
}
