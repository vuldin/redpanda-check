package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// CoreBalancing validates that core balancing on core count change is enabled.
func CoreBalancing(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "core_balancing",
		Description: "Core balancing on core count change enabled",
		Level:       checker.LevelRecommended,
	}

	config, err := pc.ClusterConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get cluster config: %v", err)
		pc.AddResult(r)
		return
	}

	if boolFromConfig(config, "core_balancing_on_core_count_change") {
		r.Status = checker.StatusPass
		r.Details = "core_balancing_on_core_count_change is enabled"
	} else {
		r.Status = checker.StatusWarn
		r.Details = "core_balancing_on_core_count_change is disabled"
	}
	pc.AddResult(r)
}
