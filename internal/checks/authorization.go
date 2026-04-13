package checks

import (
	"context"
	"fmt"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// Authorization validates that authorization is enabled.
func Authorization(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "authorization_enabled",
		Description: "Authorization enabled",
		Level:       checker.LevelCritical,
	}

	config, err := pc.ClusterConfig(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get cluster config: %v", err)
		pc.AddResult(r)
		return
	}

	kafkaAuth := boolFromConfig(config, "kafka_enable_authorization")
	sasl := boolFromConfig(config, "enable_sasl")

	if !kafkaAuth && !sasl {
		r.Status = checker.StatusFail
		r.Details = "Authorization is disabled (enable kafka_enable_authorization or enable_sasl)"
	} else {
		r.Status = checker.StatusPass
		switch {
		case kafkaAuth && sasl:
			r.Details = "Authorization enabled (kafka_enable_authorization and enable_sasl)"
		case kafkaAuth:
			r.Details = "Authorization enabled (kafka_enable_authorization)"
		default:
			r.Details = "Authorization enabled (enable_sasl)"
		}
	}
	pc.AddResult(r)
}
