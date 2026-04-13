package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// ExistingTopicsReplication validates all existing topics have replication
// factor >=3. This check uses the admin API's cluster partition listing.
func ExistingTopicsReplication(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "existing_topics_replication",
		Description: "Existing topics replication factor (>=3)",
		Level:       checker.LevelCritical,
	}

	// Use the admin API to get all cluster partitions with their replica counts.
	partitions, err := pc.AdminClient.AllClusterPartitions(ctx, false, false)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to list cluster partitions: %v", err)
		pc.AddResult(r)
		return
	}

	// Build a map of topic -> max replica count across partitions.
	topicReplicas := make(map[string]int)
	for _, p := range partitions {
		key := p.Ns + "/" + p.Topic
		replicas := len(p.Replicas)
		if existing, ok := topicReplicas[key]; !ok || replicas < existing {
			topicReplicas[key] = replicas
		}
	}

	var low []string
	for topic, replicas := range topicReplicas {
		if replicas < 3 {
			low = append(low, fmt.Sprintf("%s (replicas: %d)", topic, replicas))
		}
	}

	if len(low) > 0 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Topics with replication factor < 3: %s", strings.Join(low, ", "))
	} else if len(topicReplicas) == 0 {
		r.Status = checker.StatusPass
		r.Details = "No user topics found"
	} else {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("All %d topics have replication factor >= 3", len(topicReplicas))
	}
	pc.AddResult(r)
}
