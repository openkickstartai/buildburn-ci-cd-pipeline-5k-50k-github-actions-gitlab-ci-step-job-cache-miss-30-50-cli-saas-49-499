package main

import (
	"sort"
	"strings"
	"time"
)

// GitHub Actions per-minute pricing
const (
	RateLinux   = 0.008
	RateWindows = 0.016
	RateMacOS   = 0.08
)

// WorkflowAnalysis represents analyzed workflow run data with computed durations.
type WorkflowAnalysis struct {
	Name string
	Jobs []JobAnalysis
}

// JobAnalysis holds a single job's duration and metadata.
type JobAnalysis struct {
	Name     string
	Duration time.Duration
	Labels   []string
	Steps    []StepAnalysis
}

// StepAnalysis holds a single step's duration.
type StepAnalysis struct {
	Name     string
	Duration time.Duration
}

// StepCostEntry represents the cost attribution for a single step.
type StepCostEntry struct {
	Workflow string  `json:"workflow"`
	Job      string  `json:"job"`
	Step     string  `json:"step"`
	Cost     float64 `json:"cost"`
	Minutes  float64 `json:"minutes"`
}

// CostReport contains the full cost breakdown.
type CostReport struct {
	PerStep         []StepCostEntry    `json:"per_step"`
	PerJob          map[string]float64 `json:"per_job"`
	PerWorkflow     map[string]float64 `json:"per_workflow"`
	TotalCost       float64            `json:"total_cost"`
	ProjectedCost30 float64            `json:"projected_cost_30day"`
	TopSteps        []StepCostEntry    `json:"top_steps"`
}

// rateForOS returns the per-minute rate for a given OS label string.
func rateForOS(osLabel string) float64 {
	low := strings.ToLower(osLabel)
	if strings.Contains(low, "macos") {
		return RateMacOS
	}
	if strings.Contains(low, "windows") {
		return RateWindows
	}
	return RateLinux
}

// resolveRate determines the per-minute rate from job labels or a default OS.
func resolveRate(labels []string, defaultOS string) float64 {
	if len(labels) > 0 {
		return RunnerRate(labels)
	}
	return rateForOS(defaultOS)
}

// CalculateCost computes a full CostReport from analyzed workflow data.
// analyses: slice of WorkflowAnalysis from AnalyzeWorkflowRuns output.
// defaultOS: runner OS label used when a job has no labels (e.g. "linux", "windows", "macos").
// observationDays: number of days the input data spans (used for 30-day projection).
func CalculateCost(analyses []WorkflowAnalysis, defaultOS string, observationDays int) CostReport {
	if observationDays <= 0 {
		observationDays = 1
	}

	report := CostReport{
		PerJob:      make(map[string]float64),
		PerWorkflow: make(map[string]float64),
	}

	var allSteps []StepCostEntry

	for _, wf := range analyses {
		var wfCost float64
		for _, job := range wf.Jobs {
			rate := resolveRate(job.Labels, defaultOS)
			jobMinutes := job.Duration.Minutes()
			if jobMinutes <= 0 {
				continue
			}
			jobCost := jobMinutes * rate

			report.PerJob[job.Name] += jobCost
			wfCost += jobCost

			for _, step := range job.Steps {
				stepMinutes := step.Duration.Minutes()
				if stepMinutes <= 0 {
					continue
				}
				stepCost := stepMinutes * rate
				entry := StepCostEntry{
					Workflow: wf.Name,
					Job:      job.Name,
					Step:     step.Name,
					Cost:     stepCost,
					Minutes:  stepMinutes,
				}
				allSteps = append(allSteps, entry)
			}
		}
		report.PerWorkflow[wf.Name] += wfCost
		report.TotalCost += wfCost
	}

	report.PerStep = allSteps

	// Sort all steps by cost descending and pick top 5
	sorted := make([]StepCostEntry, len(allSteps))
	copy(sorted, allSteps)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Cost > sorted[j].Cost
	})
	top := 5
	if len(sorted) < top {
		top = len(sorted)
	}
	report.TopSteps = sorted[:top]

	// 30-day projected cost
	dailyCost := report.TotalCost / float64(observationDays)
	report.ProjectedCost30 = dailyCost * 30

	return report
}
