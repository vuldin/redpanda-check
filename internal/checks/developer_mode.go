package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// DeveloperMode validates developer mode is disabled. developer_mode is a
// node-level property, so we check each broker's node config individually.
func DeveloperMode(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "developer_mode",
		Description: "Developer mode disabled",
		Level:       checker.LevelCritical,
	}

	configs, unreachable, err := pc.PerBrokerNodeConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get node config: %v", err)
		pc.AddResult(r)
		return
	}

	var enabled []string
	for brokerID, nc := range configs {
		if boolFromConfig(nc, "developer_mode") {
			enabled = append(enabled, brokerID)
		}
	}

	if len(enabled) > 0 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Developer mode is enabled on brokers: %s", strings.Join(enabled, ", "))
	} else {
		r.Status = checker.StatusPass
		detail := "Developer mode is disabled"
		if unreachable > 0 {
			detail += fmt.Sprintf(" (%d brokers checked via single connection)", unreachable)
		}
		r.Details = detail
	}
	pc.AddResult(r)
}
