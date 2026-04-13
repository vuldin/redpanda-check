package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// BallastFile validates that a ballast file is configured on all brokers.
// A ballast file provides emergency disk space recovery by being deletable
// when the disk is full.
func BallastFile(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "ballast_file",
		Description: "Ballast file configured",
		Level:       checker.LevelRecommended,
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
		path, _ := nc["ballast_file_path"].(string)
		size, _ := nc["ballast_file_size"].(string)
		if path == "" && size == "" {
			missing = append(missing, nodeID)
		}
	}

	checked := len(configs)
	if len(missing) > 0 {
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("Ballast file not configured on brokers: %s", strings.Join(missing, ", "))
	} else {
		detail := fmt.Sprintf("Ballast file configured on %d/%d brokers checked", checked, len(brokers))
		if unreachable > 0 {
			detail += fmt.Sprintf(" (%d brokers checked via single connection)", unreachable)
		}
		r.Status = checker.StatusPass
		r.Details = detail
	}
	pc.AddResult(r)
}
