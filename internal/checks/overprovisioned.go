package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// Overprovisioned validates overprovisioned mode is disabled.
func Overprovisioned(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "overprovisioned",
		Description: "Overprovisioned mode disabled",
		Level:       checker.LevelCritical,
	}

	config, err := pc.ClusterConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get cluster config: %v", err)
		pc.AddResult(r)
		return
	}

	if boolFromConfig(config, "overprovisioned") {
		r.Status = checker.StatusFail
		r.Details = "Overprovisioned mode is enabled (must be disabled for production)"
	} else {
		r.Status = checker.StatusPass
		r.Details = "Overprovisioned mode is disabled"
	}
	pc.AddResult(r)
}
