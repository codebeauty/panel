package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClaudeBuildInvocation(t *testing.T) {
	a := NewClaudeAdapter("/usr/local/bin/claude", nil)
	inv := a.BuildInvocation(RunParams{
		PromptFile: "/tmp/out/prompt.md",
		WorkDir:    "/tmp/project",
		ReadOnly:   ReadOnlyEnforced,
	})

	assert.Equal(t, "/usr/local/bin/claude", inv.Binary)
	assert.Contains(t, inv.Args, "-p")
	assert.Contains(t, inv.Args, "--output-format")
	assert.Contains(t, inv.Args, "text")
	assert.Contains(t, inv.Args, "--tools")
	assert.Contains(t, inv.Args, "--allowedTools")
	assert.Contains(t, inv.Args, "--strict-mcp-config")
	// Last arg is the instruction to read the prompt file
	lastArg := inv.Args[len(inv.Args)-1]
	assert.Contains(t, lastArg, "/tmp/out/prompt.md")
	assert.Equal(t, "/tmp/project", inv.Dir)
	assert.Empty(t, inv.Stdin)
}

func TestClaudeBuildInvocationNoReadOnly(t *testing.T) {
	a := NewClaudeAdapter("/usr/local/bin/claude", nil)
	inv := a.BuildInvocation(RunParams{
		PromptFile: "/tmp/out/prompt.md",
		ReadOnly:   ReadOnlyNone,
	})

	assert.NotContains(t, inv.Args, "--tools")
	assert.NotContains(t, inv.Args, "--allowedTools")
}
