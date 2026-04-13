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
