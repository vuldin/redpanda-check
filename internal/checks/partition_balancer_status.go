package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// PartitionBalancerStatus validates the partition balancer is not stalled.
func PartitionBalancerStatus(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "partition_balancer_status",
		Description: "Partition balancer not stalled",
		Level:       checker.LevelRecommended,
	}

	status, err := pc.AdminClient.GetPartitionStatus(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get partition balancer status: %v", err)
		pc.AddResult(r)
		return
	}

	switch status.Status {
	case "ready":
		r.Status = checker.StatusPass
		r.Details = "Partition balancer is ready (nothing to do)"
	case "off":
		r.Status = checker.StatusPass
		r.Details = "Partition balancer is off"
	case "in_progress":
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("Partition balancer is in progress (%d reassignments)",
			status.CurrentReassignmentsCount)
	case "starting":
		r.Status = checker.StatusPass
		r.Details = "Partition balancer is starting"
	case "stalled":
		r.Status = checker.StatusWarn
		r.Details = "Partition balancer is stalled (cannot make progress on violations)"
	default:
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("Partition balancer status: %s", status.Status)
	}
	pc.AddResult(r)
}
