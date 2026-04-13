package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// DeveloperMode validates developer mode is disabled.
func DeveloperMode(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "developer_mode",
		Description: "Developer mode disabled",
		Level:       checker.LevelCritical,
	}

	config, err := pc.ClusterConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get cluster config: %v", err)
		pc.AddResult(r)
		return
	}

	if boolFromConfig(config, "developer_mode") {
		r.Status = checker.StatusFail
		r.Details = "Developer mode is enabled (must be disabled for production)"
	} else {
		r.Status = checker.StatusPass
		r.Details = "Developer mode is disabled"
	}
	pc.AddResult(r)
}
