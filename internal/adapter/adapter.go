package adapter

import "time"

type ReadOnlyMode string

const (
	ReadOnlyEnforced   ReadOnlyMode = "enforced"
	ReadOnlyBestEffort ReadOnlyMode = "bestEffort"
	ReadOnlyNone       ReadOnlyMode = "none"
)

type RunParams struct {
	Prompt     string
	PromptFile string // path to prompt file (written by runner before dispatch)
	WorkDir    string
	ReadOnly   ReadOnlyMode
	Timeout    time.Duration
	Env        []string
}

type Cost struct {
	InputTokens  int     `json:"inputTokens,omitempty"`
	OutputTokens int     `json:"outputTokens,omitempty"`
	TotalUSD     float64 `json:"totalUsd,omitempty"`
}

// Invocation describes how to launch a tool process.
type Invocation struct {
	Binary string
	Args   []string
	Stdin  string // if non-empty, piped to the process's stdin
	Dir    string
}

// Adapter builds invocations for a specific AI CLI tool.
type Adapter interface {
	Name() string
	BuildInvocation(params RunParams) Invocation
	ParseCost(stderr []byte) Cost
}
