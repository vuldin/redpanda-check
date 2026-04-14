package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// SASLUsers validates that at least one SASL user is configured when
// authorization is enabled. Having auth enabled with no users means no
// clients can authenticate.
func SASLUsers(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "sasl_users",
		Description: "SASL users configured",
		Level:       checker.LevelCritical,
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

	users, err := pc.AdminClient.ListUsers(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to list SASL users: %v", err)
		pc.AddResult(r)
		return
	}

	if len(users) == 0 {
		r.Status = checker.StatusFail
		r.Details = "Authorization is enabled but no SASL users are configured"
	} else {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("%d SASL user(s) configured", len(users))
	}
	pc.AddResult(r)
}
