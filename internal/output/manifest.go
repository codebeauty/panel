package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/codebeauty/panel/internal/runner"
)

type Manifest struct {
	Version     int              `json:"version"`
	Prompt      string           `json:"prompt"`
	StartedAt   time.Time        `json:"startedAt"`
	CompletedAt time.Time        `json:"completedAt"`
	Duration    string           `json:"duration"`
	Platform    string           `json:"platform"`
	Config      ManifestConfig   `json:"config"`
	Results     []ManifestResult `json:"results"`
}

type ManifestConfig struct {
	ReadOnly    string `json:"readOnly"`
	Timeout     int    `json:"timeout"`
	MaxParallel int    `json:"maxParallel"`
}

type ManifestResult struct {
	ToolID     string      `json:"toolId"`
	Status     string      `json:"status"`
	Duration   string      `json:"duration"`
	ExitCode   int         `json:"exitCode"`
	OutputFile string      `json:"outputFile"`
	StderrFile string      `json:"stderrFile"`
	Cost       *runner.Cost `json:"cost,omitempty"`
	Expert     string       `json:"expert,omitempty"`
}

func ReadManifest(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing run.json: %w", err)
	}
	return &m, nil
}

func WriteManifest(dir string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return AtomicWrite(filepath.Join(dir, "run.json"), data, 0o600)
}

func BuildManifest(prompt string, startedAt time.Time, results []runner.Result, cfg ManifestConfig) *Manifest {
	completedAt := time.Now()
	mResults := make([]ManifestResult, len(results))
	for i, r := range results {
		mr := ManifestResult{
			ToolID:     r.ToolID,
			Status:     string(r.Status),
			Duration:   r.Duration.Round(time.Millisecond).String(),
			ExitCode:   r.ExitCode,
			OutputFile: r.ToolID + ".md",
			StderrFile: r.ToolID + ".stderr",
		}
		if r.Cost.TotalUSD > 0 || r.Cost.InputTokens > 0 {
			cost := r.Cost
			mr.Cost = &cost
		}
		mResults[i] = mr
	}

	return &Manifest{
		Version:     1,
		Prompt:      prompt,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		Duration:    completedAt.Sub(startedAt).Round(time.Millisecond).String(),
		Platform:    runtime.GOOS + "/" + runtime.GOARCH,
		Config:      cfg,
		Results:     mResults,
	}
}
