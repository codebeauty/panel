package adapter

import "fmt"

type CursorAdapter struct {
	binary     string
	extraFlags []string
}

func NewCursorAdapter(binary string, extraFlags []string) *CursorAdapter {
	return &CursorAdapter{binary: binary, extraFlags: extraFlags}
}

func (a *CursorAdapter) Name() string { return "cursor-agent" }

func (a *CursorAdapter) BuildInvocation(p RunParams) Invocation {
	args := []string{"-p", "--output-format", "text", "--trust"}

	if p.ReadOnly != ReadOnlyNone {
		args = append(args, "--mode", "ask")
	}

	args = append(args, a.extraFlags...)

	instruction := fmt.Sprintf("Read the file at %s and follow the instructions within it.", p.PromptFile)
	args = append(args, instruction)

	return Invocation{
		Binary: a.binary,
		Args:   args,
		Dir:    p.WorkDir,
	}
}

func (a *CursorAdapter) ParseCost(stderr []byte) Cost { return Cost{} }

func init() {
	register("cursor-agent", func() Adapter {
		return NewCursorAdapter("cursor-agent", nil)
	})
}
