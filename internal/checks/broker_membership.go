package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// BrokerMembership validates all brokers are active (not decommissioning).
func BrokerMembership(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "broker_membership",
		Description: "All brokers active (not decommissioning)",
		Level:       checker.LevelCritical,
	}

	brokers, err := pc.BrokerList(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to list brokers: %v", err)
		pc.AddResult(r)
		return
	}

	var inactive []string
	for _, b := range brokers {
		if string(b.MembershipStatus) != "active" {
			inactive = append(inactive, fmt.Sprintf("%d (%s)", b.NodeID, b.MembershipStatus))
		}
	}

	if len(inactive) > 0 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Brokers not active: %s", strings.Join(inactive, ", "))
	} else {
		r.Status = checker.StatusPass
		r.Details = "All brokers are active"
	}
	pc.AddResult(r)
}
