package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
)

// EnterpriseLicense validates that enterprise features are not being used
// without a valid license.
func EnterpriseLicense(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "enterprise_license_compliance",
		Description: "No enterprise features used without valid license",
		Level:       checker.LevelCritical,
	}

	resp, err := pc.AdminClient.GetEnterpriseFeatures(ctx)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("Unable to get enterprise features: %v", err)
		pc.AddResult(r)
		return
	}

	var enabledFeatures []string
	for _, f := range resp.Features {
		if f.Enabled {
			enabledFeatures = append(enabledFeatures, f.Name)
		}
	}

	if resp.Violation {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("License violation: enterprise features in use without valid license (status: %s, features: %s)",
			resp.LicenseStatus, strings.Join(enabledFeatures, ", "))
	} else if len(enabledFeatures) > 0 {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("Enterprise features in use with valid license: %s", strings.Join(enabledFeatures, ", "))
	} else {
		r.Status = checker.StatusPass
		r.Details = "No enterprise features in use"
	}
	pc.AddResult(r)
}
