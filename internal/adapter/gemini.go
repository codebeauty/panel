package adapter

var geminiReadOnlyTools = []string{
	"read_file", "list_directory", "search_file_content",
	"glob", "google_web_search", "codebase_investigator",
}

type GeminiAdapter struct {
	binary     string
	extraFlags []string
}

func NewGeminiAdapter(binary string, extraFlags []string) *GeminiAdapter {
	return &GeminiAdapter{binary: binary, extraFlags: extraFlags}
}

func (a *GeminiAdapter) Name() string { return "gemini" }

func (a *GeminiAdapter) BuildInvocation(p RunParams) Invocation {
	args := []string{"-p", ""}

	args = append(args, a.extraFlags...)

	if p.ReadOnly != ReadOnlyNone {
		args = append(args, "--extensions", "")
		for _, tool := range geminiReadOnlyTools {
			args = append(args, "--allowed-tools", tool)
		}
	}

	args = append(args, "--output-format", "text")

	prompt := p.Prompt + "\n\nIMPORTANT: Do not narrate or describe the tools you are using. Go straight to the answer."

	return Invocation{
		Binary: a.binary,
		Args:   args,
		Stdin:  prompt,
		Dir:    p.WorkDir,
	}
}

func (a *GeminiAdapter) ParseCost(stderr []byte) Cost { return Cost{} }

func init() {
	register("gemini", func() Adapter {
		return NewGeminiAdapter("gemini", nil)
	})
}
