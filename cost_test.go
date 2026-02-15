package main

import (
	"math"
	"testing"
	"time"
)

func dur(mins float64) time.Duration {
	return time.Duration(mins * float64(time.Minute))
}

func approxEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

// Test 1: Linux pricing ($0.008/min)
func TestCalculateCost_LinuxPricing(t *testing.T) {
	analyses := []WorkflowAnalysis{
		{
			Name: "CI",
			Jobs: []JobAnalysis{
				{
					Name:     "build",
					Duration: dur(10),
					Labels:   []string{"ubuntu-latest"},
					Steps: []StepAnalysis{
						{Name: "checkout", Duration: dur(1)},
						{Name: "compile", Duration: dur(9)},
					},
				},
			},
		},
	}
	report := CalculateCost(analyses, "linux", 7)

	expectedJobCost := 10 * RateLinux // 0.08
	if !approxEqual(report.TotalCost, expectedJobCost, 0.001) {
		t.Errorf("TotalCost = %f, want %f", report.TotalCost, expectedJobCost)
	}
	if !approxEqual(report.PerJob["build"], expectedJobCost, 0.001) {
		t.Errorf("PerJob[build] = %f, want %f", report.PerJob["build"], expectedJobCost)
	}
	if !approxEqual(report.PerWorkflow["CI"], expectedJobCost, 0.001) {
		t.Errorf("PerWorkflow[CI] = %f, want %f", report.PerWorkflow["CI"], expectedJobCost)
	}
	if len(report.PerStep) != 2 {
		t.Fatalf("expected 2 step entries, got %d", len(report.PerStep))
	}
	// Verify step costs
	checkoutCost := 1 * RateLinux
	compileCost := 9 * RateLinux
	if !approxEqual(report.PerStep[0].Cost, checkoutCost, 0.001) {
		t.Errorf("PerStep[0].Cost = %f, want %f", report.PerStep[0].Cost, checkoutCost)
	}
	if !approxEqual(report.PerStep[1].Cost, compileCost, 0.001) {
		t.Errorf("PerStep[1].Cost = %f, want %f", report.PerStep[1].Cost, compileCost)
	}
}

// Test 2: Windows pricing ($0.016/min)
func TestCalculateCost_WindowsPricing(t *testing.T) {
	analyses := []WorkflowAnalysis{
		{
			Name: "CI-Win",
			Jobs: []JobAnalysis{
				{
					Name:     "build-win",
					Duration: dur(10),
					Labels:   []string{"windows-latest"},
					Steps: []StepAnalysis{
						{Name: "checkout", Duration: dur(2)},
						{Name: "compile", Duration: dur(8)},
					},
				},
			},
		},
	}
	report := CalculateCost(analyses, "linux", 7)

	expectedCost := 10 * RateWindows // 0.16
	if !approxEqual(report.TotalCost, expectedCost, 0.001) {
		t.Errorf("TotalCost = %f, want %f", report.TotalCost, expectedCost)
	}
	if !approxEqual(report.PerJob["build-win"], expectedCost, 0.001) {
		t.Errorf("PerJob[build-win] = %f, want %f", report.PerJob["build-win"], expectedCost)
	}
	expectedStepCost := 2 * RateWindows
	if !approxEqual(report.PerStep[0].Cost, expectedStepCost, 0.001) {
		t.Errorf("PerStep[0].Cost = %f, want %f", report.PerStep[0].Cost, expectedStepCost)
	}
}

// Test 3: macOS pricing ($0.08/min)
func TestCalculateCost_MacOSPricing(t *testing.T) {
	analyses := []WorkflowAnalysis{
		{
			Name: "CI-Mac",
			Jobs: []JobAnalysis{
				{
					Name:     "build-mac",
					Duration: dur(5),
					Labels:   []string{"macos-latest"},
					Steps: []StepAnalysis{
						{Name: "xcode-build", Duration: dur(5)},
					},
				},
			},
		},
	}
	report := CalculateCost(analyses, "linux", 1)

	expectedCost := 5 * RateMacOS // 0.40
	if !approxEqual(report.TotalCost, expectedCost, 0.001) {
		t.Errorf("TotalCost = %f, want %f", report.TotalCost, expectedCost)
	}
	// 30-day projection from 1 day observation
	expectedProjection := expectedCost * 30
	if !approxEqual(report.ProjectedCost30, expectedProjection, 0.01) {
		t.Errorf("ProjectedCost30 = %f, want %f", report.ProjectedCost30, expectedProjection)
	}
}

// Test 4: Zero-duration edge case
func TestCalculateCost_ZeroDuration(t *testing.T) {
	analyses := []WorkflowAnalysis{
		{
			Name: "CI",
			Jobs: []JobAnalysis{
				{
					Name:     "noop",
					Duration: 0,
					Labels:   []string{"ubuntu-latest"},
					Steps: []StepAnalysis{
						{Name: "nothing", Duration: 0},
					},
				},
			},
		},
	}
	report := CalculateCost(analyses, "linux", 7)

	if report.TotalCost != 0 {
		t.Errorf("TotalCost = %f, want 0", report.TotalCost)
	}
	if len(report.PerStep) != 0 {
		t.Errorf("expected 0 step entries for zero duration, got %d", len(report.PerStep))
	}
	if len(report.TopSteps) != 0 {
		t.Errorf("expected 0 top steps for zero duration, got %d", len(report.TopSteps))
	}
	if report.ProjectedCost30 != 0 {
		t.Errorf("ProjectedCost30 = %f, want 0", report.ProjectedCost30)
	}
	// PerWorkflow entry should exist but be zero
	if report.PerWorkflow["CI"] != 0 {
		t.Errorf("PerWorkflow[CI] = %f, want 0", report.PerWorkflow["CI"])
	}
}

// Test 5: Multi-workflow aggregation with all 3 OS tiers
func TestCalculateCost_MultiWorkflowAggregation(t *testing.T) {
	analyses := []WorkflowAnalysis{
		{
			Name: "CI",
			Jobs: []JobAnalysis{
				{
					Name:     "build",
					Duration: dur(10),
					Labels:   []string{"ubuntu-latest"},
					Steps: []StepAnalysis{
						{Name: "checkout", Duration: dur(1)},
						{Name: "compile", Duration: dur(9)},
					},
				},
			},
		},
		{
			Name: "Deploy",
			Jobs: []JobAnalysis{
				{
					Name:     "deploy-prod",
					Duration: dur(5),
					Labels:   []string{"macos-latest"},
					Steps: []StepAnalysis{
						{Name: "deploy", Duration: dur(5)},
					},
				},
			},
		},
		{
			Name: "Lint",
			Jobs: []JobAnalysis{
				{
					Name:     "lint",
					Duration: dur(2),
					Labels:   []string{"windows-latest"},
					Steps: []StepAnalysis{
						{Name: "eslint", Duration: dur(2)},
					},
				},
			},
		},
	}
	report := CalculateCost(analyses, "linux", 7)

	expectedTotal := 10*RateLinux + 5*RateMacOS + 2*RateWindows // 0.08 + 0.40 + 0.032 = 0.512
	if !approxEqual(report.TotalCost, expectedTotal, 0.001) {
		t.Errorf("TotalCost = %f, want %f", report.TotalCost, expectedTotal)
	}

	// Verify per-workflow breakdown
	if !approxEqual(report.PerWorkflow["CI"], 10*RateLinux, 0.001) {
		t.Errorf("PerWorkflow[CI] = %f, want %f", report.PerWorkflow["CI"], 10*RateLinux)
	}
	if !approxEqual(report.PerWorkflow["Deploy"], 5*RateMacOS, 0.001) {
		t.Errorf("PerWorkflow[Deploy] = %f, want %f", report.PerWorkflow["Deploy"], 5*RateMacOS)
	}
	if !approxEqual(report.PerWorkflow["Lint"], 2*RateWindows, 0.001) {
		t.Errorf("PerWorkflow[Lint] = %f, want %f", report.PerWorkflow["Lint"], 2*RateWindows)
	}

	// Verify per-job breakdown
	if !approxEqual(report.PerJob["build"], 10*RateLinux, 0.001) {
		t.Errorf("PerJob[build] = %f, want %f", report.PerJob["build"], 10*RateLinux)
	}
	if !approxEqual(report.PerJob["deploy-prod"], 5*RateMacOS, 0.001) {
		t.Errorf("PerJob[deploy-prod] = %f, want %f", report.PerJob["deploy-prod"], 5*RateMacOS)
	}

	// Top steps: most expensive should be deploy (5 * 0.08 = 0.40)
	if len(report.TopSteps) != 4 {
		t.Fatalf("expected 4 top steps (all steps since < 5), got %d", len(report.TopSteps))
	}
	if report.TopSteps[0].Step != "deploy" {
		t.Errorf("TopSteps[0].Step = %s, want deploy", report.TopSteps[0].Step)
	}
	if !approxEqual(report.TopSteps[0].Cost, 5*RateMacOS, 0.001) {
		t.Errorf("TopSteps[0].Cost = %f, want %f", report.TopSteps[0].Cost, 5*RateMacOS)
	}

	// 30-day projection from 7 days
	expectedProjection := (expectedTotal / 7) * 30
	if !approxEqual(report.ProjectedCost30, expectedProjection, 0.01) {
		t.Errorf("ProjectedCost30 = %f, want %f", report.ProjectedCost30, expectedProjection)
	}
}

// Test 6: Top-5 step limit with more than 5 steps
func TestCalculateCost_TopStepsLimit(t *testing.T) {
	steps := []StepAnalysis{
		{Name: "step1", Duration: dur(1)},
		{Name: "step2", Duration: dur(2)},
		{Name: "step3", Duration: dur(3)},
		{Name: "step4", Duration: dur(4)},
		{Name: "step5", Duration: dur(5)},
		{Name: "step6", Duration: dur(6)},
		{Name: "step7", Duration: dur(7)},
	}
	analyses := []WorkflowAnalysis{
		{
			Name: "BigWorkflow",
			Jobs: []JobAnalysis{
				{
					Name:     "big-job",
					Duration: dur(28),
					Labels:   []string{"ubuntu-latest"},
					Steps:    steps,
				},
			},
		},
	}
	report := CalculateCost(analyses, "linux", 1)

	if len(report.TopSteps) != 5 {
		t.Fatalf("expected 5 top steps, got %d", len(report.TopSteps))
	}
	// All 7 steps should be in PerStep
	if len(report.PerStep) != 7 {
		t.Errorf("expected 7 per-step entries, got %d", len(report.PerStep))
	}
	// Most expensive step should be step7 (7 min * 0.008 = 0.056)
	if report.TopSteps[0].Step != "step7" {
		t.Errorf("TopSteps[0].Step = %s, want step7", report.TopSteps[0].Step)
	}
	if !approxEqual(report.TopSteps[0].Cost, 7*RateLinux, 0.001) {
		t.Errorf("TopSteps[0].Cost = %f, want %f", report.TopSteps[0].Cost, 7*RateLinux)
	}
	// Least expensive in top-5 should be step3 (3 min * 0.008 = 0.024)
	if report.TopSteps[4].Step != "step3" {
		t.Errorf("TopSteps[4].Step = %s, want step3", report.TopSteps[4].Step)
	}
}

// Test 7: Default OS fallback when job has no labels
func TestCalculateCost_DefaultOSFallback(t *testing.T) {
	analyses := []WorkflowAnalysis{
		{
			Name: "CI",
			Jobs: []JobAnalysis{
				{
					Name:     "build",
					Duration: dur(10),
					Labels:   nil, // no labels — use default
				},
			},
		},
	}

	// macOS default
	report := CalculateCost(analyses, "macos", 1)
	expected := 10 * RateMacOS
	if !approxEqual(report.TotalCost, expected, 0.001) {
		t.Errorf("TotalCost with macos default = %f, want %f", report.TotalCost, expected)
	}

	// windows default
	report = CalculateCost(analyses, "windows", 1)
	expected = 10 * RateWindows
	if !approxEqual(report.TotalCost, expected, 0.001) {
		t.Errorf("TotalCost with windows default = %f, want %f", report.TotalCost, expected)
	}

	// linux default (explicit)
	report = CalculateCost(analyses, "linux", 1)
	expected = 10 * RateLinux
	if !approxEqual(report.TotalCost, expected, 0.001) {
		t.Errorf("TotalCost with linux default = %f, want %f", report.TotalCost, expected)
	}
}

// Test 8: Zero observation days defaults to 1
func TestCalculateCost_ZeroObservationDays(t *testing.T) {
	analyses := []WorkflowAnalysis{
		{
			Name: "CI",
			Jobs: []JobAnalysis{
				{
					Name:     "build",
					Duration: dur(10),
					Labels:   []string{"ubuntu-latest"},
				},
			},
		},
	}

	// observationDays=0 should be treated as 1
	report := CalculateCost(analyses, "linux", 0)
	expectedCost := 10 * RateLinux
	if !approxEqual(report.TotalCost, expectedCost, 0.001) {
		t.Errorf("TotalCost = %f, want %f", report.TotalCost, expectedCost)
	}
	expectedProjection := expectedCost * 30 // daily cost * 30
	if !approxEqual(report.ProjectedCost30, expectedProjection, 0.01) {
		t.Errorf("ProjectedCost30 = %f, want %f", report.ProjectedCost30, expectedProjection)
	}

	// negative days also defaults to 1
	report2 := CalculateCost(analyses, "linux", -5)
	if !approxEqual(report2.ProjectedCost30, expectedProjection, 0.01) {
		t.Errorf("ProjectedCost30 with negative days = %f, want %f", report2.ProjectedCost30, expectedProjection)
	}
}
