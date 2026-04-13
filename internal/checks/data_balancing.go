package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// DataBalancing validates continuous data balancing is enabled.
func DataBalancing(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "continuous_data_balancing",
		Description: "Continuous data balancing enabled",
		Level:       checker.LevelRecommended,
	}

	config, err := pc.ClusterConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get cluster config: %v", err)
		pc.AddResult(r)
		return
	}

	mode := stringFromConfig(config, "partition_autobalancing_mode")
	if mode == "continuous" {
		r.Status = checker.StatusPass
		r.Details = "partition_autobalancing_mode is continuous"
	} else if mode == "" {
		r.Status = checker.StatusWarn
		r.Details = "partition_autobalancing_mode is not set"
	} else {
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("partition_autobalancing_mode is %q (recommend: continuous)", mode)
	}
	pc.AddResult(r)
}
