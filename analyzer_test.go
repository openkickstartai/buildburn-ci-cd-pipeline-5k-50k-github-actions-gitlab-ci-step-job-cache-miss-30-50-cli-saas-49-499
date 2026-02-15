package main

import (
	"testing"
	"time"
)

func mkJob(name, wf, conclusion string, labels []string, mins float64, steps []Step) Job {
	s := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return Job{
		Name: name, Workflow: wf, Conclusion: conclusion,
		Labels: labels, StartedAt: s,
		CompletedAt: s.Add(time.Duration(mins * float64(time.Minute))),
		Steps:       steps,
	}
}

func mkStep(name string, mins float64, conclusion string) Step {
	s := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return Step{
		Name: name, Conclusion: conclusion, StartedAt: s,
		CompletedAt: s.Add(time.Duration(mins * float64(time.Minute))),
	}
}

func TestRunnerRate(t *testing.T) {
	cases := []struct {
		labels []string
		want   float64
	}{
		{[]string{"ubuntu-latest"}, 0.008},
		{[]string{"ubuntu-22.04"}, 0.008},
		{[]string{"macos-latest"}, 0.08},
		{[]string{"macos-13"}, 0.08},
		{[]string{"windows-latest"}, 0.016},
		{[]string{"self-hosted", "linux"}, 0.008},
		{[]string{}, 0.008},
	}
	for _, c := range cases {
		if got := RunnerRate(c.labels); got != c.want {
			t.Errorf("RunnerRate(%v) = %f, want %f", c.labels, got, c.want)
		}
	}
}

func TestAnalyzeCostCalculation(t *testing.T) {
	jobs := []Job{
		mkJob("build", "CI", "success", []string{"ubuntu-latest"}, 10, nil),
		mkJob("test", "CI", "success", []string{"macos-latest"}, 5, nil),
		mkJob("lint", "Lint", "success", []string{"windows-latest"}, 2, nil),
	}
	r := Analyze(jobs, 7)
	expected := 10*0.008 + 5*0.08 + 2*0.016 // 0.08 + 0.40 + 0.032 = 0.512
	if r.TotalCost < expected-0.01 || r.TotalCost > expected+0.01 {
		t.Errorf("TotalCost = %.4f, want ~%.4f", r.TotalCost, expected)
	}
	if r.TotalMinutes != 17 {
		t.Errorf("TotalMinutes = %.0f, want 17", r.TotalMinutes)
	}
	if r.MonthlyCost < 2.0 || r.MonthlyCost > 2.5 {
		t.Errorf("MonthlyCost = %.2f, want ~2.19", r.MonthlyCost)
	}
	if len(r.ByWorkflow) != 2 {
		t.Errorf("Expected 2 workflows, got %d", len(r.ByWorkflow))
	}
}

func TestAnalyzeWasteDetection(t *testing.T) {
	jobs := []Job{
		mkJob("build", "CI", "failure", []string{"ubuntu-latest"}, 8, []Step{
			mkStep("Restore npm cache", 0.5, "failure"),
			mkStep("npm install", 5, "success"),
		}),
	}
	r := Analyze(jobs, 7)
	retryFound, cacheFound, depsFound := false, false, false
	for _, w := range r.Waste {
		switch w.Type {
		case "retry":
			retryFound = true
		case "cache-miss":
			cacheFound = true
		case "slow-deps":
			depsFound = true
		}
	}
	if !retryFound {
		t.Error("Expected retry waste")
	}
	if !cacheFound {
		t.Error("Expected cache-miss waste")
	}
	if !depsFound {
		t.Error("Expected slow-deps waste")
	}
	if len(r.Suggestions) == 0 {
		t.Error("Expected at least one suggestion")
	}
}

func TestAnalyzeEmptyJobs(t *testing.T) {
	r := Analyze(nil, 7)
	if r.TotalCost != 0 || r.TotalMinutes != 0 {
		t.Error("Empty jobs should produce zero cost")
	}
	if r.ByWorkflow == nil {
		t.Error("ByWorkflow should be initialized")
	}
}

func TestAnalyzeMacOSSuggestion(t *testing.T) {
	jobs := []Job{
		mkJob("ios", "Mobile", "success", []string{"macos-13"}, 20, nil),
	}
	r := Analyze(jobs, 7)
	hasMacTip := false
	for _, s := range r.Suggestions {
		if len(s) > 5 {
			hasMacTip = true
		}
	}
	if !hasMacTip {
		t.Error("Expected macOS cost suggestion")
	}
}
