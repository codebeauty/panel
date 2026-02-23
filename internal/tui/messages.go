package tui

import (
	"time"

	"github.com/codebeauty/panel/internal/runner"
)

type Phase int

const (
	PhaseSelect  Phase = iota // tool multi-select
	PhaseExpert               // expert picker
	PhaseConfirm              // review & dispatch
	PhaseProgress             // live execution
	PhaseSummary              // results viewer
)

type ToolStartedMsg struct {
	ToolID string
}

type ToolCompletedMsg struct {
	ToolID string
	Result runner.Result
}

type AllCompletedMsg struct {
	Results []runner.Result
	RunDir  string
}

type ErrorMsg struct {
	Err error
}

type doDispatchMsg struct{}

type ToolProgress struct {
	Status   string // pending, running, success, failed, timeout, cancelled
	Started  time.Time
	Duration time.Duration
	Words    int
}
