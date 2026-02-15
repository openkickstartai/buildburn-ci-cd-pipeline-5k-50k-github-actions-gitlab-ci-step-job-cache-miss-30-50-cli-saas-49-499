package main

import (
	"fmt"
	"sort"
	"strings"
)

type Report struct {
	TotalMinutes float64            `json:"total_minutes"`
	TotalCost    float64            `json:"total_cost"`
	MonthlyCost  float64            `json:"monthly_estimate"`
	ByWorkflow   map[string]float64 `json:"cost_by_workflow"`
	TopSteps     []StepCost         `json:"top_steps"`
	Waste        []WasteItem        `json:"waste"`
	Suggestions  []string           `json:"suggestions"`
}

type StepCost struct {
	Name string  `json:"name"`
	Min  float64 `json:"minutes"`
}

type WasteItem struct {
	Type   string  `json:"type"`
	Detail string  `json:"detail"`
	Cost   float64 `json:"cost_impact"`
}

func RunnerRate(labels []string) float64 {
	for _, l := range labels {
		low := strings.ToLower(l)
		if strings.Contains(low, "macos") {
			return 0.08
		}
		if strings.Contains(low, "windows") {
			return 0.016
		}
	}
	return 0.008
}

func Analyze(jobs []Job, days int) Report {
	r := Report{ByWorkflow: map[string]float64{}}
	stepAgg := map[string]float64{}
	hasMac := false
	for _, j := range jobs {
		rate := RunnerRate(j.Labels)
		dur := j.CompletedAt.Sub(j.StartedAt).Minutes()
		if dur <= 0 {
			continue
		}
		cost := dur * rate
		r.TotalMinutes += dur
		r.TotalCost += cost
		r.ByWorkflow[j.Workflow] += cost
		if rate == 0.08 {
			hasMac = true
		}
		if j.Conclusion == "failure" {
			r.Waste = append(r.Waste, WasteItem{"retry", "Failed: " + j.Name, cost})
		}
		for _, s := range j.Steps {
			sd := s.CompletedAt.Sub(s.StartedAt).Minutes()
			if sd <= 0 {
				continue
			}
			stepAgg[s.Name] += sd
			sn := strings.ToLower(s.Name)
			if strings.Contains(sn, "cache") && s.Conclusion == "failure" {
				r.Waste = append(r.Waste, WasteItem{"cache-miss", s.Name, sd * rate})
			}
			hasInstall := strings.Contains(sn, "install") || strings.Contains(sn, "npm") || strings.Contains(sn, "pip")
			if sd > 3 && hasInstall {
				r.Waste = append(r.Waste, WasteItem{"slow-deps", fmt.Sprintf("%s (%.1fm)", s.Name, sd), sd * rate * 0.5})
			}
		}
	}
	for n, m := range stepAgg {
		r.TopSteps = append(r.TopSteps, StepCost{n, m})
	}
	sort.Slice(r.TopSteps, func(i, j int) bool { return r.TopSteps[i].Min > r.TopSteps[j].Min })
	if len(r.TopSteps) > 10 {
		r.TopSteps = r.TopSteps[:10]
	}
	wasteSum := 0.0
	for _, w := range r.Waste {
		wasteSum += w.Cost
	}
	if wasteSum > 0 {
		r.Suggestions = append(r.Suggestions, fmt.Sprintf("Fix %d issues to save ~$%.2f/week", len(r.Waste), wasteSum))
	}
	if hasMac {
		r.Suggestions = append(r.Suggestions, "macOS runners cost 10x Linux — switch where possible")
	}
	for _, s := range r.TopSteps {
		if s.Min > 30 {
			r.Suggestions = append(r.Suggestions, fmt.Sprintf("'%s' used %.0fm — cache or parallelize", s.Name, s.Min))
		}
	}
	if days > 0 {
		r.MonthlyCost = r.TotalCost / float64(days) * 30
	}
	return r
}

func PrintReport(r Report, days int) {
	fmt.Printf("\n\U0001F525 BuildBurn Report (last %d days)\n", days)
	fmt.Println("══════════════════════════════════════════")
	fmt.Printf("  CI minutes: %.0f | Cost: $%.2f | Monthly: $%.2f\n", r.TotalMinutes, r.TotalCost, r.MonthlyCost)
	if len(r.ByWorkflow) > 0 {
		fmt.Println("\n📊 Cost by Workflow:")
		for wf, c := range r.ByWorkflow {
			pct := 0.0
			if r.TotalCost > 0 {
				pct = c / r.TotalCost * 100
			}
			fmt.Printf("  %-35s $%7.2f (%4.0f%%)\n", wf, c, pct)
		}
	}
	if len(r.TopSteps) > 0 {
		fmt.Println("\n⏱️  Top Steps:")
		for _, s := range r.TopSteps {
			fmt.Printf("  %-35s %7.1f min\n", s.Name, s.Min)
		}
	}
	if len(r.Waste) > 0 {
		fmt.Println("\n🗑️  Waste Detected:")
		for _, w := range r.Waste {
			fmt.Printf("  [%-9s] %-28s $%.2f\n", w.Type, w.Detail, w.Cost)
		}
	}
	if len(r.Suggestions) > 0 {
		fmt.Println("\n💡 Suggestions:")
		for i, s := range r.Suggestions {
			fmt.Printf("  %d. %s\n", i+1, s)
		}
	}
	fmt.Println()
}
