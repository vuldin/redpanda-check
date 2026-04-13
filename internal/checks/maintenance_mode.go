package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// MaintenanceMode validates no brokers are in maintenance mode.
func MaintenanceMode(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "maintenance_mode",
		Description: "No brokers in maintenance mode",
		Level:       checker.LevelCritical,
	}

	brokers, err := pc.BrokerList(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to list brokers: %v", err)
		pc.AddResult(r)
		return
	}

	var draining []string
	for _, b := range brokers {
		if b.Maintenance != nil && b.Maintenance.Draining {
			draining = append(draining, fmt.Sprintf("%d", b.NodeID))
		}
	}

	if len(draining) > 0 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Brokers in maintenance mode: %s", strings.Join(draining, ", "))
	} else {
		r.Status = checker.StatusPass
		r.Details = "No brokers in maintenance mode"
	}
	pc.AddResult(r)
}
