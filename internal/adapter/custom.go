package adapter

import "strings"

type CustomAdapter struct {
	name       string
	binary     string
	extraFlags []string
	useStdin   bool
}

func NewCustomAdapter(name, binary string, extraFlags []string, useStdin bool) *CustomAdapter {
	return &CustomAdapter{name: name, binary: binary, extraFlags: extraFlags, useStdin: useStdin}
}

func (a *CustomAdapter) Name() string { return a.name }

func (a *CustomAdapter) BuildInvocation(p RunParams) Invocation {
	args := make([]string, len(a.extraFlags))
	hasPlaceholder := false
	for i, flag := range a.extraFlags {
		if strings.Contains(flag, "{prompt}") {
			hasPlaceholder = true
		}
		args[i] = strings.ReplaceAll(flag, "{prompt}", p.Prompt)
	}

	inv := Invocation{
		Binary: a.binary,
		Args:   args,
		Dir:    p.WorkDir,
	}

	if a.useStdin {
		inv.Stdin = p.Prompt
	} else if !hasPlaceholder {
		inv.Args = append(inv.Args, p.Prompt)
	}

	return inv
}

func (a *CustomAdapter) ParseCost(stderr []byte) Cost { return Cost{} }
