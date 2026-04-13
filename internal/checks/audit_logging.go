package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// AuditLogging validates that audit logging is enabled.
func AuditLogging(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "audit_logging",
		Description: "Audit logging enabled",
		Level:       checker.LevelRecommended,
	}

	config, err := pc.ClusterConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get cluster config: %v", err)
		pc.AddResult(r)
		return
	}

	if boolFromConfig(config, "audit_enabled") {
		r.Status = checker.StatusPass
		r.Details = "Audit logging is enabled"
	} else {
		r.Status = checker.StatusWarn
		r.Details = "Audit logging is not enabled (audit_enabled: false)"
	}
	pc.AddResult(r)
}
