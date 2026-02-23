package adapter

type CodexAdapter struct {
	binary     string
	extraFlags []string
}

func NewCodexAdapter(binary string, extraFlags []string) *CodexAdapter {
	return &CodexAdapter{binary: binary, extraFlags: extraFlags}
}

func (a *CodexAdapter) Name() string { return "codex" }

func (a *CodexAdapter) BuildInvocation(p RunParams) Invocation {
	args := []string{"exec"}

	if p.ReadOnly != ReadOnlyNone {
		args = append(args, "--sandbox", "read-only")
	}

	args = append(args, "-c", "web_search=live", "--skip-git-repo-check")
	args = append(args, a.extraFlags...)
	args = append(args, PromptFileInstruction(p.PromptFile))

	return Invocation{
		Binary: a.binary,
		Args:   args,
		Dir:    p.WorkDir,
	}
}

func (a *CodexAdapter) ParseCost(stderr []byte) Cost { return Cost{} }

func init() {
	register("codex", func() Adapter {
		return NewCodexAdapter("codex", nil)
	})
}
