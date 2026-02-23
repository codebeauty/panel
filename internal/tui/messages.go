package tui

import (
	"time"

	"github.com/codebeauty/panel/internal/runner"
)

// Phase represents the current TUI phase.
type Phase int

const (
	PhaseSelect  Phase = iota // tool multi-select
	PhaseExpert               // expert picker
	PhaseConfirm              // review & dispatch
	PhaseProgress             // live execution
	PhaseSummary              // results viewer
)

// Messages sent from runner goroutine to BubbleTea.
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

// doDispatchMsg is sent from Init to trigger dispatch via Update,
// ensuring context/cancel are set on the model without data races.
type doDispatchMsg struct{}

// ToolProgress tracks per-tool execution state.
type ToolProgress struct {
	Status   string // pending, running, success, failed, timeout, cancelled
	Started  time.Time
	Duration time.Duration
	Words    int
}
