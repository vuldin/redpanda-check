package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// ReplicationFactor validates the default topic replication factor is >=3.
func ReplicationFactor(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "replication_factor",
		Description: "Default topic replication factor (>=3)",
		Level:       checker.LevelCritical,
	}

	config, err := pc.ClusterConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get cluster config: %v", err)
		pc.AddResult(r)
		return
	}

	rf := intFromConfig(config, "default_topic_replications")

	if rf < 3 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Default replication factor is %d (minimum: 3)", rf)
	} else {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("Default replication factor is %d", rf)
	}
	pc.AddResult(r)
}
