package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// Superusers validates that at least one superuser is configured when
// authorization is enabled.
func Superusers(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "superusers_configured",
		Description: "Superusers configured",
		Level:       checker.LevelRecommended,
	}

	config, err := pc.ClusterConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get cluster config: %v", err)
		pc.AddResult(r)
		return
	}

	authEnabled := boolFromConfig(config, "kafka_enable_authorization") || boolFromConfig(config, "enable_sasl")
	if !authEnabled {
		r.Status = checker.StatusSkip
		r.Details = "Authorization is not enabled"
		pc.AddResult(r)
		return
	}

	superusers, ok := config["superusers"].([]any)
	if !ok || len(superusers) == 0 {
		r.Status = checker.StatusWarn
		r.Details = "Authorization is enabled but no superusers are configured"
	} else {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("%d superuser(s) configured", len(superusers))
	}
	pc.AddResult(r)
}
