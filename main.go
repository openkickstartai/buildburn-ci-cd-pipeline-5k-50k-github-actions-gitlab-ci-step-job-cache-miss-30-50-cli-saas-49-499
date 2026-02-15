package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

type RunsResp struct {
	Runs []Run `json:"workflow_runs"`
}

type Run struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type JobsResp struct {
	Jobs []Job `json:"jobs"`
}

type Job struct {
	Name        string    `json:"name"`
	Workflow    string    `json:"-"`
	Conclusion  string    `json:"conclusion"`
	Labels      []string  `json:"labels"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Steps       []Step    `json:"steps"`
}

type Step struct {
	Name        string    `json:"name"`
	Conclusion  string    `json:"conclusion"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

func ghGet(url, token string, out interface{}) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func main() {
	repo := flag.String("repo", "", "GitHub repo (owner/repo)")
	token := flag.String("token", "", "GitHub token (or GITHUB_TOKEN env)")
	days := flag.Int("days", 7, "Lookback window in days")
	outFmt := flag.String("format", "table", "Output: table|json")
	flag.Parse()
	if *repo == "" {
		fmt.Fprintln(os.Stderr, "Usage: buildburn -repo owner/repo [-token TOKEN] [-days 7]")
		os.Exit(1)
	}
	tk := *token
	if tk == "" {
		tk = os.Getenv("GITHUB_TOKEN")
	}
	if tk == "" {
		fmt.Fprintln(os.Stderr, "Error: set -token flag or GITHUB_TOKEN env")
		os.Exit(1)
	}
	since := time.Now().AddDate(0, 0, -*days).Format("2006-01-02")
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs?created=>=%s&per_page=100", *repo, since)
	var rr RunsResp
	if err := ghGet(apiURL, tk, &rr); err != nil {
		fmt.Fprintf(os.Stderr, "Fetch error: %v\n", err)
		os.Exit(1)
	}
	var allJobs []Job
	for _, run := range rr.Runs {
		var jr JobsResp
		u := fmt.Sprintf("https://api.github.com/repos/%s/actions/runs/%d/jobs?per_page=100", *repo, run.ID)
		if ghGet(u, tk, &jr) == nil {
			for i := range jr.Jobs {
				jr.Jobs[i].Workflow = run.Name
				allJobs = append(allJobs, jr.Jobs[i])
			}
		}
	}
	report := Analyze(allJobs, *days)
	if *outFmt == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(report)
	} else {
		PrintReport(report, *days)
	}
}
