package checker_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vuldin/redpanda-check/internal/checker"
)

func TestNewReport(t *testing.T) {
	results := []checker.CheckResult{
		{Name: "a", Status: checker.StatusPass, Level: checker.LevelCritical},
		{Name: "b", Status: checker.StatusFail, Level: checker.LevelCritical},
		{Name: "c", Status: checker.StatusWarn, Level: checker.LevelRecommended},
		{Name: "d", Status: checker.StatusSkip, Level: checker.LevelCritical},
	}
	report := checker.NewReport(results)

	if report.OverallStatus != checker.StatusFail {
		t.Errorf("expected overall FAIL, got %s", report.OverallStatus)
	}
	if report.Summary.Total != 4 {
		t.Errorf("expected total 4, got %d", report.Summary.Total)
	}
	if report.Summary.Passed != 1 {
		t.Errorf("expected 1 passed, got %d", report.Summary.Passed)
	}
	if report.Summary.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", report.Summary.Failed)
	}
}

func TestNewReport_AllPass(t *testing.T) {
	results := []checker.CheckResult{
		{Name: "a", Status: checker.StatusPass},
		{Name: "b", Status: checker.StatusPass},
	}
	report := checker.NewReport(results)

	if report.OverallStatus != checker.StatusPass {
		t.Errorf("expected overall PASS, got %s", report.OverallStatus)
	}
}

func TestPrintText_Default(t *testing.T) {
	results := []checker.CheckResult{
		{Name: "pass_check", Description: "Passes", Status: checker.StatusPass},
		{Name: "fail_check", Description: "Fails", Status: checker.StatusFail, Details: "something broke"},
	}
	report := checker.NewReport(results)

	var buf bytes.Buffer
	checker.PrintText(&buf, report, false, false)
	out := buf.String()

	// Non-verbose: should not show passing check.
	if strings.Contains(out, "Passes") {
		t.Error("non-verbose output should not contain passing check description")
	}
	if !strings.Contains(out, "Fails") {
		t.Error("output should contain failing check description")
	}
	if !strings.Contains(out, "something broke") {
		t.Error("output should contain failure details")
	}
	if !strings.Contains(out, "Overall: FAIL") {
		t.Error("output should contain overall status")
	}
}

func TestPrintText_Verbose(t *testing.T) {
	results := []checker.CheckResult{
		{Name: "pass_check", Description: "Passes", Status: checker.StatusPass},
	}
	report := checker.NewReport(results)

	var buf bytes.Buffer
	checker.PrintText(&buf, report, true, false)
	out := buf.String()

	if !strings.Contains(out, "Passes") {
		t.Error("verbose output should contain passing check")
	}
}

func TestPrintJSON(t *testing.T) {
	results := []checker.CheckResult{
		{Name: "a", Status: checker.StatusPass, Level: checker.LevelCritical},
	}
	report := checker.NewReport(results)

	var buf bytes.Buffer
	if err := checker.PrintJSON(&buf, report); err != nil {
		t.Fatalf("PrintJSON error: %v", err)
	}

	var decoded checker.Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("unable to unmarshal JSON output: %v", err)
	}
	if decoded.OverallStatus != checker.StatusPass {
		t.Errorf("expected PASS in JSON, got %s", decoded.OverallStatus)
	}
	if len(decoded.Checks) != 1 {
		t.Errorf("expected 1 check in JSON, got %d", len(decoded.Checks))
	}
}
