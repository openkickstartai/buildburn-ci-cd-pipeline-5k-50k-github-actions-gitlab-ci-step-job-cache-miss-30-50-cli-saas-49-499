package main

import (
	"fmt"
	"strings"
)

// CostReport holds the data needed for recommendation analysis.
// Defined locally to avoid blocking dependency on cost.go.
type CostReport struct {
	Jobs []Job
	Days int
}

// Recommendation represents a single optimization suggestion.
type Recommendation struct {
	RuleID                  string  `json:"rule_id"`
	Severity                string  `json:"severity"`
	EstimatedMonthlySavings float64 `json:"estimated_monthly_savings"`
	Fix                     string  `json:"fix"`
}

// GenerateRecommendations runs all detection rules against the given CostReport
// and returns a combined list of optimization suggestions.
func GenerateRecommendations(cr CostReport) []Recommendation {
	var recs []Recommendation
	recs = append(recs, DetectCacheMiss(cr)...)
	recs = append(recs, DetectLongCheckout(cr)...)
	recs = append(recs, DetectDuplicateSteps(cr)...)
	recs = append(recs, DetectExpensiveOS(cr)...)
	return recs
}

// DetectCacheMiss flags steps whose names contain dependency-install keywords
// (install, npm ci, pip install, go mod download) and that exceed 60 seconds.
// It suggests adding a cache action to speed up these steps.
func DetectCacheMiss(cr CostReport) []Recommendation {
	keywords := []string{"install", "npm ci", "pip install", "go mod download"}
	var recs []Recommendation
	for _, j := range cr.Jobs {
		rate := RunnerRate(j.Labels)
		for _, s := range j.Steps {
			durSec := s.CompletedAt.Sub(s.StartedAt).Seconds()
			if durSec <= 60 {
				continue
			}
			sn := strings.ToLower(s.Name)
			matched := false
			for _, kw := range keywords {
				if strings.Contains(sn, kw) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
			// Estimate: caching saves ~70% of install time
			savedMin := (durSec * 0.7) / 60.0
			scale := monthlyScale(cr.Days)
			monthlySavings := savedMin * rate * scale

			recs = append(recs, Recommendation{
				RuleID:                  "cache-miss",
				Severity:                "high",
				EstimatedMonthlySavings: monthlySavings,
				Fix:                     fmt.Sprintf("Add actions/cache for step '%s' (took %.0fs). Caching dependencies can reduce this step by ~70%%.", s.Name, durSec),
			})
		}
	}
	return recs
}

// DetectLongCheckout flags checkout steps that exceed 30 seconds and suggests
// using shallow clone (fetch-depth: 1) to reduce checkout time.
func DetectLongCheckout(cr CostReport) []Recommendation {
	var recs []Recommendation
	for _, j := range cr.Jobs {
		rate := RunnerRate(j.Labels)
		for _, s := range j.Steps {
			durSec := s.CompletedAt.Sub(s.StartedAt).Seconds()
			if durSec <= 30 {
				continue
			}
			sn := strings.ToLower(s.Name)
			if !strings.Contains(sn, "checkout") {
				continue
			}
			// Estimate: shallow clone saves ~50% of checkout time
			savedMin := (durSec * 0.5) / 60.0
			scale := monthlyScale(cr.Days)
			monthlySavings := savedMin * rate * scale

			recs = append(recs, Recommendation{
				RuleID:                  "long-checkout",
				Severity:                "low",
				EstimatedMonthlySavings: monthlySavings,
				Fix:                     fmt.Sprintf("Use shallow clone (fetch-depth: 1) for step '%s' (took %.0fs). This can cut checkout time by ~50%%.", s.Name, durSec),
			})
		}
	}
	return recs
}

// DetectDuplicateSteps flags identical step names that appear across multiple
// distinct jobs, suggesting the use of a matrix strategy or reusable workflow
// to reduce duplication.
func DetectDuplicateSteps(cr CostReport) []Recommendation {
	// Map step name -> set of job names that contain it
	stepJobs := map[string]map[string]bool{}
	for _, j := range cr.Jobs {
		for _, s := range j.Steps {
			if stepJobs[s.Name] == nil {
				stepJobs[s.Name] = map[string]bool{}
			}
			stepJobs[s.Name][j.Name] = true
		}
	}

	var recs []Recommendation
	for stepName, jobs := range stepJobs {
		if len(jobs) < 2 {
			continue
		}
		// Estimate: consolidation saves ~$5/month per duplicated job
		recs = append(recs, Recommendation{
			RuleID:                  "duplicate-steps",
			Severity:                "medium",
			EstimatedMonthlySavings: float64(len(jobs)-1) * 5.0,
			Fix:                     fmt.Sprintf("Step '%s' appears in %d jobs. Consider using a matrix strategy or reusable workflow to reduce duplication.", stepName, len(jobs)),
		})
	}
	return recs
}

// DetectExpensiveOS flags macOS jobs that run longer than 5 minutes and suggests
// moving non-GUI tests to Linux runners to save ~90%% on runner costs.
func DetectExpensiveOS(cr CostReport) []Recommendation {
	var recs []Recommendation
	for _, j := range cr.Jobs {
		isMac := false
		for _, l := range j.Labels {
			if strings.Contains(strings.ToLower(l), "macos") {
				isMac = true
				break
			}
		}
		if !isMac {
			continue
		}
		durMin := j.CompletedAt.Sub(j.StartedAt).Minutes()
		if durMin <= 5 {
			continue
		}
		// macOS rate is 0.08/min, Linux is 0.008/min — saving is 0.072/min
		scale := monthlyScale(cr.Days)
		monthlySavings := durMin * (0.08 - 0.008) * scale

		recs = append(recs, Recommendation{
			RuleID:                  "expensive-os",
			Severity:                "high",
			EstimatedMonthlySavings: monthlySavings,
			Fix:                     fmt.Sprintf("Job '%s' runs on macOS for %.1f min ($%.2f/run). macOS runners cost 10x Linux. Move non-GUI tests to ubuntu-latest to save ~90%%.", j.Name, durMin, durMin*0.08),
		})
	}
	return recs
}

// monthlyScale returns a multiplier to extrapolate observed data to a monthly estimate.
func monthlyScale(days int) float64 {
	if days <= 0 {
		return 1
	}
	return 30.0 / float64(days)
}
