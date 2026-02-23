package adapter

import "fmt"

const readOnlyTools = "Read,Glob,Grep,WebFetch,WebSearch"

type ClaudeAdapter struct {
	binary     string
	extraFlags []string
}

func NewClaudeAdapter(binary string, extraFlags []string) *ClaudeAdapter {
	return &ClaudeAdapter{binary: binary, extraFlags: extraFlags}
}

func (a *ClaudeAdapter) Name() string { return "claude" }

func (a *ClaudeAdapter) BuildInvocation(p RunParams) Invocation {
	args := []string{"-p", "--output-format", "text"}

	args = append(args, a.extraFlags...)

	if p.ReadOnly != ReadOnlyNone {
		args = append(args,
			"--tools", readOnlyTools,
			"--allowedTools", readOnlyTools,
			"--strict-mcp-config",
		)
	}

	instruction := fmt.Sprintf("Read the file at %s and follow the instructions within it.", p.PromptFile)
	args = append(args, instruction)

	return Invocation{
		Binary: a.binary,
		Args:   args,
		Dir:    p.WorkDir,
	}
}

func (a *ClaudeAdapter) ParseCost(stderr []byte) Cost {
	return Cost{}
}

func init() {
	register("claude", func() Adapter {
		return NewClaudeAdapter("claude", nil)
	})
}
