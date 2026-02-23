package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodexBuildInvocation(t *testing.T) {
	a := NewCodexAdapter("/usr/local/bin/codex", nil)
	inv := a.BuildInvocation(RunParams{
		PromptFile: "/tmp/out/prompt.md",
		WorkDir:    "/tmp/project",
		ReadOnly:   ReadOnlyEnforced,
	})
	assert.Equal(t, "exec", inv.Args[0])
	assert.Contains(t, inv.Args, "--sandbox")
	assert.Contains(t, inv.Args, "read-only")
	assert.Contains(t, inv.Args, "--skip-git-repo-check")
	lastArg := inv.Args[len(inv.Args)-1]
	assert.Contains(t, lastArg, "prompt.md")
	assert.Empty(t, inv.Stdin)
}

func TestGeminiBuildInvocation(t *testing.T) {
	a := NewGeminiAdapter("/usr/local/bin/gemini", nil)
	inv := a.BuildInvocation(RunParams{
		Prompt:   "review this code",
		ReadOnly: ReadOnlyEnforced,
	})
	assert.Contains(t, inv.Args, "--output-format")
	assert.Contains(t, inv.Args, "--extensions")
	assert.Contains(t, inv.Args, "--allowed-tools")
	assert.NotEmpty(t, inv.Stdin)
	assert.Contains(t, inv.Stdin, "review this code")
}

func TestGeminiBuildInvocationNoReadOnly(t *testing.T) {
	a := NewGeminiAdapter("/usr/local/bin/gemini", nil)
	inv := a.BuildInvocation(RunParams{
		Prompt:   "hello",
		ReadOnly: ReadOnlyNone,
	})
	assert.NotContains(t, inv.Args, "--allowed-tools")
}

func TestAmpBuildInvocation(t *testing.T) {
	a := NewAmpAdapter("/usr/local/bin/amp", nil)
	inv := a.BuildInvocation(RunParams{
		Prompt: "review this",
	})
	assert.Contains(t, inv.Args, "-x")
	assert.NotEmpty(t, inv.Stdin)
	assert.Contains(t, inv.Stdin, "review this")
}

func TestCustomBuildInvocation(t *testing.T) {
	a := NewCustomAdapter("mytool", "/usr/local/bin/mytool",
		[]string{"--query", "{prompt}", "--format", "markdown"}, false)
	inv := a.BuildInvocation(RunParams{
		Prompt: "review this",
	})
	assert.Contains(t, inv.Args, "--query")
	assert.Contains(t, inv.Args, "review this")
	assert.NotContains(t, inv.Args, "{prompt}")
}

func TestCursorBuildInvocation(t *testing.T) {
	a := NewCursorAdapter("/usr/local/bin/cursor-agent", []string{"--model", "opus-4.6-thinking"})
	inv := a.BuildInvocation(RunParams{
		PromptFile: "/tmp/out/prompt.md",
		WorkDir:    "/tmp/project",
		ReadOnly:   ReadOnlyEnforced,
	})
	assert.Contains(t, inv.Args, "-p")
	assert.Contains(t, inv.Args, "--output-format")
	assert.Contains(t, inv.Args, "--trust")
	assert.Contains(t, inv.Args, "--mode")
	assert.Contains(t, inv.Args, "ask")
	assert.Contains(t, inv.Args, "--model")
	assert.Contains(t, inv.Args, "opus-4.6-thinking")
	lastArg := inv.Args[len(inv.Args)-1]
	assert.Contains(t, lastArg, "prompt.md")
	assert.Empty(t, inv.Stdin)
}

func TestCursorBuildInvocationNoReadOnly(t *testing.T) {
	a := NewCursorAdapter("/usr/local/bin/cursor-agent", nil)
	inv := a.BuildInvocation(RunParams{
		PromptFile: "/tmp/out/prompt.md",
		WorkDir:    "/tmp/project",
		ReadOnly:   ReadOnlyNone,
	})
	assert.NotContains(t, inv.Args, "--mode")
	assert.NotContains(t, inv.Args, "ask")
}

func TestCustomBuildInvocationStdin(t *testing.T) {
	a := NewCustomAdapter("mytool", "/usr/local/bin/mytool",
		[]string{"--format", "markdown"}, true)
	inv := a.BuildInvocation(RunParams{
		Prompt: "review this",
	})
	assert.Equal(t, "review this", inv.Stdin)
	assert.NotContains(t, inv.Args, "review this")
}
