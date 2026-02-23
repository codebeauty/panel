package adapter

import (
	"fmt"
	"time"
)

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

type Invocation struct {
	Binary string
	Args   []string
	Stdin  string // if non-empty, piped to the process's stdin
	Dir    string
}

type Adapter interface {
	Name() string
	BuildInvocation(params RunParams) Invocation
	ParseCost(stderr []byte) Cost
}

// PromptFileInstruction returns the standard instruction that tells an AI CLI
// to read the prompt from a file.
func PromptFileInstruction(promptFile string) string {
	return fmt.Sprintf("Read the file at %s and follow the instructions within it.", promptFile)
}
