package adapter

type AmpAdapter struct {
	binary     string
	extraFlags []string
}

func NewAmpAdapter(binary string, extraFlags []string) *AmpAdapter {
	return &AmpAdapter{binary: binary, extraFlags: extraFlags}
}

func (a *AmpAdapter) Name() string { return "amp" }

func (a *AmpAdapter) BuildInvocation(p RunParams) Invocation {
	args := []string{"-x"}
	args = append(args, a.extraFlags...)

	return Invocation{
		Binary: a.binary,
		Args:   args,
		Stdin:  p.Prompt,
		Dir:    p.WorkDir,
	}
}

func (a *AmpAdapter) ParseCost(stderr []byte) Cost { return Cost{} }

func init() {
	register("amp", func() Adapter {
		return NewAmpAdapter("amp", nil)
	})
}
