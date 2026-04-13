package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// MinTopicReplications validates that minimum_topic_replications is set to at
// least 3, preventing users from creating under-replicated topics.
func MinTopicReplications(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "min_topic_replications",
		Description: "Minimum topic replications >= 3",
		Level:       checker.LevelCritical,
	}

	config, err := pc.ClusterConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get cluster config: %v", err)
		pc.AddResult(r)
		return
	}

	val := intFromConfig(config, "minimum_topic_replications")
	if val >= 3 {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("minimum_topic_replications is %d", val)
	} else {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("minimum_topic_replications is %d (minimum: 3)", val)
	}
	pc.AddResult(r)
}
