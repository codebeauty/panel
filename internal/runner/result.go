package runner

import (
	"time"

	"github.com/codebeauty/panel/internal/adapter"
)

type Status string

const (
	StatusSuccess   Status = "success"
	StatusFailed    Status = "failed"
	StatusTimeout   Status = "timeout"
	StatusCancelled Status = "cancelled"
)

type Cost = adapter.Cost

type Result struct {
	ToolID   string        `json:"toolId"`
	Status   Status        `json:"status"`
	Stdout   []byte        `json:"-"`
	Stderr   []byte        `json:"-"`
	Duration time.Duration `json:"duration"`
	Cost     Cost          `json:"cost,omitempty"`
	ExitCode int           `json:"exitCode"`
}
