package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// VersionRecency warns when the cluster is running N-2 (two minor releases
// behind latest) — still supported but approaching EOL. VersionConsistency
// handles older-than-N-2 as a Critical failure and current-or-N-1 as PASS.
func VersionRecency(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "version_recency",
		Description: "Redpanda version is current (not approaching EOL)",
		Level:       checker.LevelRecommended,
	}

	brokers, err := pc.BrokerList(ctx)
	if err != nil {
		r.Status = checker.StatusSkip
		r.Details = fmt.Sprintf("Unable to list brokers: %v", err)
		pc.AddResult(r)
		return
	}

	if len(brokers) == 0 {
		r.Status = checker.StatusSkip
		r.Details = "No brokers available"
		pc.AddResult(r)
		return
	}

	// VersionConsistency already reports inconsistency as a failure; here we
	// just inspect the first broker's version to grade recency. If the
	// version is unparseable we SKIP (the Critical check will have FAILed).
	parsed, err := parseRedpandaVersion(brokers[0].Version)
	if err != nil {
		r.Status = checker.StatusSkip
		r.Details = "Version format is not a valid release (see version_consistency)"
		pc.AddResult(r)
		return
	}

	offset := minorOffsetFromLatest(parsed)
	switch {
	case offset <= 1:
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("%s is the current or most recent prior release", parsed)
	case offset == 2:
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf(
			"%s is two minor releases behind latest (v%d.%d) and approaching EOL; plan an upgrade",
			parsed, redpandaLatestMajor, redpandaLatestMinor)
	default:
		// Older than N-2 is already a Critical failure in VersionConsistency.
		r.Status = checker.StatusSkip
		r.Details = fmt.Sprintf(
			"%s is %d minor releases behind latest; see version_consistency",
			parsed, offset)
	}
	pc.AddResult(r)
}
