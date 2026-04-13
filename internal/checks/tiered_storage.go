package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// TieredStorage validates that tiered storage (cloud storage) is enabled.
func TieredStorage(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "tiered_storage",
		Description: "Tiered storage enabled",
		Level:       checker.LevelRecommended,
	}

	config, err := pc.ClusterConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get cluster config: %v", err)
		pc.AddResult(r)
		return
	}

	if boolFromConfig(config, "cloud_storage_enabled") {
		r.Status = checker.StatusPass
		bucket := stringFromConfig(config, "cloud_storage_bucket")
		region := stringFromConfig(config, "cloud_storage_region")
		details := "Tiered storage is enabled"
		if bucket != "" {
			details += fmt.Sprintf(" (bucket: %s", bucket)
			if region != "" {
				details += fmt.Sprintf(", region: %s", region)
			}
			details += ")"
		}
		r.Details = details
	} else {
		r.Status = checker.StatusWarn
		r.Details = "Tiered storage is not enabled (cloud_storage_enabled: false)"
	}
	pc.AddResult(r)
}
