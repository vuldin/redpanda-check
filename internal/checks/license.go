package checks

import (
	"context"
	"fmt"
	"time"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// License validates that a Redpanda license is loaded.
func License(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "license_check",
		Description: "Verify Redpanda license",
		Level:       checker.LevelCritical,
	}

	license, err := pc.AdminClient.GetLicenseInfo(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to retrieve license info: %v", err)
		pc.AddResult(r)
		return
	}

	if !license.Loaded {
		r.Status = checker.StatusWarn
		r.Details = "No license loaded"
		pc.AddResult(r)
		return
	}

	props := license.Properties
	r.Status = checker.StatusPass
	r.Details = fmt.Sprintf("License loaded (type: %s, org: %s", props.Type, props.Organization)
	if props.Expires > 0 {
		exp := time.Unix(props.Expires, 0)
		r.Details += fmt.Sprintf(", expires: %s", exp.Format("2006-01-02"))
		if time.Now().After(exp) {
			r.Status = checker.StatusFail
			r.Details = fmt.Sprintf("License expired on %s", exp.Format("2006-01-02"))
			pc.AddResult(r)
			return
		}
	}
	r.Details += ")"
	pc.AddResult(r)
}

// LicenseExpiry checks whether the license is approaching expiration.
// Expiring within 7 days is a critical failure; within 30 days is a
// recommended warning.
func LicenseExpiry(ctx context.Context, pc *checker.ProductionChecker) {
	license, err := pc.AdminClient.GetLicenseInfo(ctx)
	if err != nil || !license.Loaded || license.Properties.Expires == 0 {
		// Nothing to check — the License check handles missing/errored cases.
		return
	}

	exp := time.Unix(license.Properties.Expires, 0)
	remaining := time.Until(exp)
	daysLeft := int(remaining.Hours() / 24)

	if remaining <= 7*24*time.Hour {
		pc.AddResult(checker.CheckResult{
			Name:        "license_expiry",
			Description: "License expiring within 7 days",
			Level:       checker.LevelCritical,
			Status:      checker.StatusFail,
			Details:     fmt.Sprintf("License expires in %d days (%s) — renew immediately", daysLeft, exp.Format("2006-01-02")),
		})
	} else if remaining <= 30*24*time.Hour {
		pc.AddResult(checker.CheckResult{
			Name:        "license_expiry",
			Description: "License expiring within 30 days",
			Level:       checker.LevelRecommended,
			Status:      checker.StatusWarn,
			Details:     fmt.Sprintf("License expires in %d days (%s) — plan renewal", daysLeft, exp.Format("2006-01-02")),
		})
	} else {
		pc.AddResult(checker.CheckResult{
			Name:        "license_expiry",
			Description: "License not expiring soon",
			Level:       checker.LevelRecommended,
			Status:      checker.StatusPass,
			Details:     fmt.Sprintf("License expires in %d days (%s)", daysLeft, exp.Format("2006-01-02")),
		})
	}
}
