package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// ClusterHealth validates cluster health using the admin API.
func ClusterHealth(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "cluster_health",
		Description: "Cluster health validation",
		Level:       checker.LevelCritical,
	}

	health, err := pc.AdminClient.GetHealthOverview(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to retrieve cluster health: %v", err)
		pc.AddResult(r)
		return
	}

	if health.IsHealthy {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("Cluster is healthy with %d nodes", len(health.AllNodes))
	} else {
		r.Status = checker.StatusFail
		var parts []string
		if len(health.UnhealthyReasons) > 0 {
			parts = append(parts, fmt.Sprintf("reasons: %v", health.UnhealthyReasons))
		}
		if len(health.NodesDown) > 0 {
			parts = append(parts, fmt.Sprintf("nodes down: %v", health.NodesDown))
		}
		if health.LeaderlessCount != nil && *health.LeaderlessCount > 0 {
			parts = append(parts, fmt.Sprintf("leaderless partitions: %d", *health.LeaderlessCount))
		}
		if health.UnderReplicatedCount != nil && *health.UnderReplicatedCount > 0 {
			parts = append(parts, fmt.Sprintf("under-replicated partitions: %d", *health.UnderReplicatedCount))
		}
		r.Details = "Cluster is unhealthy"
		if len(parts) > 0 {
			r.Details += " (" + strings.Join(parts, ", ") + ")"
		}
	}

	pc.AddResult(r)
}
