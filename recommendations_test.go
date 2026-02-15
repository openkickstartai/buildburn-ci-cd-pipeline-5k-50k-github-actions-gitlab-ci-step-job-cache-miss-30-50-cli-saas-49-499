package main

import (
	"testing"
	"time"
)

// Helper: create a Step with duration in seconds.
func mkRecStep(name string, secs float64, conclusion string) Step {
	s := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return Step{
		Name:        name,
		Conclusion:  conclusion,
		StartedAt:   s,
		CompletedAt: s.Add(time.Duration(secs * float64(time.Second))),
	}
}

// Helper: create a Job with duration in minutes.
func mkRecJob(name, workflow string, labels []string, mins float64, steps []Step) Job {
	s := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return Job{
		Name:        name,
		Workflow:    workflow,
		Conclusion:  "success",
		Labels:      labels,
		StartedAt:   s,
		CompletedAt: s.Add(time.Duration(mins * float64(time.Minute))),
		Steps:       steps,
	}
}

// ========================
// DetectCacheMiss tests
// ========================

func TestDetectCacheMiss_Triggers_NpmInstall(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("build", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("npm install", 90, "success"), // 90s > 60s threshold
			}),
		},
		Days: 7,
	}
	recs := DetectCacheMiss(cr)
	if len(recs) == 0 {
		t.Fatal("expected at least one recommendation for slow npm install step")
	}
	if recs[0].RuleID != "cache-miss" {
		t.Errorf("RuleID = %s, want cache-miss", recs[0].RuleID)
	}
	if recs[0].Severity != "high" {
		t.Errorf("Severity = %s, want high", recs[0].Severity)
	}
	if recs[0].EstimatedMonthlySavings <= 0 {
		t.Errorf("EstimatedMonthlySavings = %f, want > 0", recs[0].EstimatedMonthlySavings)
	}
	if recs[0].Fix == "" {
		t.Error("Fix string should not be empty")
	}
}

func TestDetectCacheMiss_NoTrigger_UnderThreshold(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("build", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("npm install", 30, "success"), // 30s <= 60s threshold
			}),
		},
		Days: 7,
	}
	recs := DetectCacheMiss(cr)
	if len(recs) != 0 {
		t.Errorf("expected no recommendations for fast install step, got %d", len(recs))
	}
}

func TestDetectCacheMiss_Triggers_GoModDownload(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("build", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("go mod download", 120, "success"),
			}),
		},
		Days: 7,
	}
	recs := DetectCacheMiss(cr)
	if len(recs) == 0 {
		t.Fatal("expected recommendation for slow go mod download")
	}
	if recs[0].RuleID != "cache-miss" {
		t.Errorf("RuleID = %s, want cache-miss", recs[0].RuleID)
	}
}

func TestDetectCacheMiss_Triggers_PipInstall(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("build", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("pip install dependencies", 75, "success"),
			}),
		},
		Days: 7,
	}
	recs := DetectCacheMiss(cr)
	if len(recs) == 0 {
		t.Fatal("expected recommendation for slow pip install step")
	}
	if recs[0].RuleID != "cache-miss" {
		t.Errorf("RuleID = %s, want cache-miss", recs[0].RuleID)
	}
}

func TestDetectCacheMiss_NoTrigger_NonInstallStep(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("build", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("Run tests", 120, "success"), // long but not an install step
			}),
		},
		Days: 7,
	}
	recs := DetectCacheMiss(cr)
	if len(recs) != 0 {
		t.Errorf("expected no recommendations for non-install step, got %d", len(recs))
	}
}

// ========================
// DetectLongCheckout tests
// ========================

func TestDetectLongCheckout_Triggers(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("build", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("Checkout code", 45, "success"), // 45s > 30s threshold
			}),
		},
		Days: 7,
	}
	recs := DetectLongCheckout(cr)
	if len(recs) == 0 {
		t.Fatal("expected recommendation for long checkout step")
	}
	if recs[0].RuleID != "long-checkout" {
		t.Errorf("RuleID = %s, want long-checkout", recs[0].RuleID)
	}
	if recs[0].Severity != "low" {
		t.Errorf("Severity = %s, want low", recs[0].Severity)
	}
	if recs[0].EstimatedMonthlySavings <= 0 {
		t.Errorf("EstimatedMonthlySavings = %f, want > 0", recs[0].EstimatedMonthlySavings)
	}
	if recs[0].Fix == "" {
		t.Error("Fix string should not be empty")
	}
}

func TestDetectLongCheckout_NoTrigger_FastCheckout(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("build", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("Checkout code", 10, "success"), // 10s <= 30s threshold
			}),
		},
		Days: 7,
	}
	recs := DetectLongCheckout(cr)
	if len(recs) != 0 {
		t.Errorf("expected no recommendations for fast checkout, got %d", len(recs))
	}
}

func TestDetectLongCheckout_NoTrigger_NonCheckoutStep(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("build", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("Build app", 120, "success"), // long but not checkout
			}),
		},
		Days: 7,
	}
	recs := DetectLongCheckout(cr)
	if len(recs) != 0 {
		t.Errorf("expected no recommendations for non-checkout step, got %d", len(recs))
	}
}

// ========================
// DetectDuplicateSteps tests
// ========================

func TestDetectDuplicateSteps_Triggers(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("build-linux", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("Run tests", 60, "success"),
			}),
			mkRecJob("build-mac", "CI", []string{"macos-latest"}, 5, []Step{
				mkRecStep("Run tests", 60, "success"), // same step name in different job
			}),
		},
		Days: 7,
	}
	recs := DetectDuplicateSteps(cr)
	if len(recs) == 0 {
		t.Fatal("expected recommendation for duplicate steps across jobs")
	}
	if recs[0].RuleID != "duplicate-steps" {
		t.Errorf("RuleID = %s, want duplicate-steps", recs[0].RuleID)
	}
	if recs[0].Severity != "medium" {
		t.Errorf("Severity = %s, want medium", recs[0].Severity)
	}
	if recs[0].EstimatedMonthlySavings <= 0 {
		t.Errorf("EstimatedMonthlySavings = %f, want > 0", recs[0].EstimatedMonthlySavings)
	}
	if recs[0].Fix == "" {
		t.Error("Fix string should not be empty")
	}
}

func TestDetectDuplicateSteps_NoTrigger_UniqueSteps(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("build", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("Run tests", 60, "success"),
			}),
			mkRecJob("deploy", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("Deploy app", 60, "success"), // different step name
			}),
		},
		Days: 7,
	}
	recs := DetectDuplicateSteps(cr)
	if len(recs) != 0 {
		t.Errorf("expected no recommendations for unique steps, got %d", len(recs))
	}
}

func TestDetectDuplicateSteps_Triggers_ThreeJobs(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("job-1", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("Setup Node", 30, "success"),
			}),
			mkRecJob("job-2", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("Setup Node", 30, "success"),
			}),
			mkRecJob("job-3", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("Setup Node", 30, "success"),
			}),
		},
		Days: 7,
	}
	recs := DetectDuplicateSteps(cr)
	if len(recs) == 0 {
		t.Fatal("expected recommendation for step appearing in 3 jobs")
	}
	// 3 jobs - 1 = 2 duplicates, $5 each = $10
	if recs[0].EstimatedMonthlySavings != 10.0 {
		t.Errorf("EstimatedMonthlySavings = %f, want 10.0", recs[0].EstimatedMonthlySavings)
	}
}

// ========================
// DetectExpensiveOS tests
// ========================

func TestDetectExpensiveOS_Triggers(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("test-mac", "CI", []string{"macos-latest"}, 10, nil), // 10min > 5min
		},
		Days: 7,
	}
	recs := DetectExpensiveOS(cr)
	if len(recs) == 0 {
		t.Fatal("expected recommendation for expensive macOS job")
	}
	if recs[0].RuleID != "expensive-os" {
		t.Errorf("RuleID = %s, want expensive-os", recs[0].RuleID)
	}
	if recs[0].Severity != "high" {
		t.Errorf("Severity = %s, want high", recs[0].Severity)
	}
	if recs[0].EstimatedMonthlySavings <= 0 {
		t.Errorf("EstimatedMonthlySavings = %f, want > 0", recs[0].EstimatedMonthlySavings)
	}
	if recs[0].Fix == "" {
		t.Error("Fix string should not be empty")
	}
}

func TestDetectExpensiveOS_NoTrigger_ShortMacJob(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("test-mac", "CI", []string{"macos-latest"}, 3, nil), // 3min <= 5min
		},
		Days: 7,
	}
	recs := DetectExpensiveOS(cr)
	if len(recs) != 0 {
		t.Errorf("expected no recommendations for short macOS job, got %d", len(recs))
	}
}

func TestDetectExpensiveOS_NoTrigger_LinuxJob(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("test-linux", "CI", []string{"ubuntu-latest"}, 10, nil), // Linux, not macOS
		},
		Days: 7,
	}
	recs := DetectExpensiveOS(cr)
	if len(recs) != 0 {
		t.Errorf("expected no recommendations for Linux job, got %d", len(recs))
	}
}

func TestDetectExpensiveOS_MonthlySavingsCalculation(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("test-mac", "CI", []string{"macos-13"}, 10, nil),
		},
		Days: 30,
	}
	recs := DetectExpensiveOS(cr)
	if len(recs) == 0 {
		t.Fatal("expected recommendation")
	}
	// 10 min * (0.08 - 0.008) * (30/30) = 10 * 0.072 * 1 = 0.72
	expected := 0.72
	if recs[0].EstimatedMonthlySavings < expected-0.01 || recs[0].EstimatedMonthlySavings > expected+0.01 {
		t.Errorf("EstimatedMonthlySavings = %f, want ~%f", recs[0].EstimatedMonthlySavings, expected)
	}
}

// ========================
// GenerateRecommendations combined test
// ========================

func TestGenerateRecommendations_Combined(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{
			mkRecJob("build", "CI", []string{"macos-latest"}, 10, []Step{
				mkRecStep("npm install", 90, "success"),
				mkRecStep("Checkout", 45, "success"),
			}),
			mkRecJob("test", "CI", []string{"ubuntu-latest"}, 5, []Step{
				mkRecStep("npm install", 90, "success"),
			}),
		},
		Days: 7,
	}
	recs := GenerateRecommendations(cr)

	// Expect: cache-miss x2, long-checkout x1, duplicate-steps x1, expensive-os x1 = 5 total
	if len(recs) < 4 {
		t.Errorf("expected at least 4 recommendations, got %d", len(recs))
	}

	ruleIDs := map[string]bool{}
	for _, r := range recs {
		ruleIDs[r.RuleID] = true
	}
	for _, expected := range []string{"cache-miss", "long-checkout", "duplicate-steps", "expensive-os"} {
		if !ruleIDs[expected] {
			t.Errorf("expected rule %s in recommendations", expected)
		}
	}
}

func TestGenerateRecommendations_Empty(t *testing.T) {
	cr := CostReport{
		Jobs: []Job{},
		Days: 7,
	}
	recs := GenerateRecommendations(cr)
	if len(recs) != 0 {
		t.Errorf("expected no recommendations for empty report, got %d", len(recs))
	}
}
