package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// VersionConsistency validates that all brokers run the same Redpanda version,
// that the version follows the proper vMAJOR.MINOR.PATCH format, and that the
// version is still within the supported window (latest, N-1, or N-2).
//
// Anything older than N-2 (the EOL window) fails as unsupported. The WARN
// outcome for N-2 "approaching EOL" is emitted by the separate
// VersionRecency check (Recommended level).
func VersionConsistency(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "version_consistency",
		Description: "Consistent, valid, and supported Redpanda version across brokers",
		Level:       checker.LevelCritical,
	}

	brokers, err := pc.BrokerList(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to list brokers: %v", err)
		pc.AddResult(r)
		return
	}

	versions := make(map[string][]string)
	for _, b := range brokers {
		versions[b.Version] = append(versions[b.Version], fmt.Sprintf("%d", b.NodeID))
	}

	if len(versions) == 0 {
		r.Status = checker.StatusSkip
		r.Details = "No version information available"
		pc.AddResult(r)
		return
	}

	if len(versions) > 1 {
		var parts []string
		for v, nodes := range versions {
			parts = append(parts, fmt.Sprintf("%s (brokers: %s)", v, strings.Join(nodes, ", ")))
		}
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Inconsistent versions: %s", strings.Join(parts, "; "))
		pc.AddResult(r)
		return
	}

	// Exactly one reported version across all brokers.
	var reported string
	for v := range versions {
		reported = v
	}

	parsed, err := parseRedpandaVersion(reported)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("%s; all brokers report %q", err.Error(), reported)
		pc.AddResult(r)
		return
	}

	offset := minorOffsetFromLatest(parsed)
	switch {
	case offset <= 1:
		// Latest or N-1.
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("All brokers running %s (current release)", parsed)
	case offset == 2:
		// N-2 is still supported — flagged separately by VersionRecency.
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf(
			"All brokers running %s (N-2, still supported but see version_recency)", parsed)
	default:
		// N-3 or older: unsupported.
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf(
			"All brokers running %s, which is %d minor releases behind latest (v%d.%d); unsupported",
			parsed, offset, redpandaLatestMajor, redpandaLatestMinor)
	}
	pc.AddResult(r)
}
