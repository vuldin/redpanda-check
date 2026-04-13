package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// BrokerCount validates minimum broker count (>=3 for production).
func BrokerCount(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "broker_count",
		Description: "Minimum broker count (>=3)",
		Level:       checker.LevelCritical,
	}

	brokers, err := pc.BrokerList(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to retrieve broker list: %v", err)
		pc.AddResult(r)
		return
	}

	n := len(brokers)
	if n >= 3 {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("Found %d brokers", n)
	} else if n > 0 {
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("Only %d brokers (recommend >=3 for production)", n)
	} else {
		r.Status = checker.StatusFail
		r.Details = "No brokers found"
	}
	pc.AddResult(r)
}
