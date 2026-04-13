package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// VersionConsistency validates all brokers run the same Redpanda version.
func VersionConsistency(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "version_consistency",
		Description: "Consistent Redpanda version across brokers",
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

	switch len(versions) {
	case 0:
		r.Status = checker.StatusSkip
		r.Details = "No version information available"
	case 1:
		for v := range versions {
			r.Status = checker.StatusPass
			r.Details = fmt.Sprintf("All brokers running version %s", v)
		}
	default:
		var parts []string
		for v, nodes := range versions {
			parts = append(parts, fmt.Sprintf("%s (brokers: %s)", v, strings.Join(nodes, ", ")))
		}
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Inconsistent versions: %s", strings.Join(parts, "; "))
	}
	pc.AddResult(r)
}
