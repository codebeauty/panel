package adapter

// Model represents a variant of an AI tool (e.g., different model sizes).
type Model struct {
	ID          string
	DisplayName string
	CompoundID  string // used as the tool config key (e.g., "claude-opus")
	ExtraFlags  []string
	Recommended bool
}

// AdapterModels maps adapter names to their available models.
var AdapterModels = map[string][]Model{
	"claude": {
		{ID: "opus", DisplayName: "Opus 4.6 — most capable", CompoundID: "claude-opus", ExtraFlags: []string{"--model", "opus"}, Recommended: true},
		{ID: "sonnet", DisplayName: "Sonnet 4.5 — fast and capable", CompoundID: "claude-sonnet", ExtraFlags: []string{"--model", "sonnet"}},
		{ID: "haiku", DisplayName: "Haiku 4.5 — fastest, most affordable", CompoundID: "claude-haiku", ExtraFlags: []string{"--model", "haiku"}},
	},
	"codex": {
		{ID: "gpt-5.3-codex", DisplayName: "GPT-5.3 Codex — high reasoning", CompoundID: "codex-5.3-high", ExtraFlags: []string{"-m", "gpt-5.3-codex", "-c", "model_reasoning_effort=high"}, Recommended: true},
		{ID: "gpt-5.3-codex", DisplayName: "GPT-5.3 Codex — xhigh reasoning", CompoundID: "codex-5.3-xhigh", ExtraFlags: []string{"-m", "gpt-5.3-codex", "-c", "model_reasoning_effort=xhigh"}},
		{ID: "gpt-5.3-codex", DisplayName: "GPT-5.3 Codex — medium reasoning", CompoundID: "codex-5.3-medium", ExtraFlags: []string{"-m", "gpt-5.3-codex", "-c", "model_reasoning_effort=medium"}},
	},
	"gemini": {
		{ID: "gemini-3.1-pro", DisplayName: "Gemini 3.1 Pro — latest", CompoundID: "gemini-3.1-pro", ExtraFlags: []string{"-m", "gemini-3.1-pro-preview"}, Recommended: true},
		{ID: "gemini-2.5-pro", DisplayName: "Gemini 2.5 Pro — stable GA", CompoundID: "gemini-2.5-pro", ExtraFlags: []string{"-m", "gemini-2.5-pro"}},
		{ID: "gemini-3-flash", DisplayName: "Gemini 3 Flash — fast", CompoundID: "gemini-3-flash", ExtraFlags: []string{"-m", "gemini-3-flash-preview"}},
		{ID: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash — fast GA", CompoundID: "gemini-2.5-flash", ExtraFlags: []string{"-m", "gemini-2.5-flash"}},
	},
	"amp": {
		{ID: "smart", DisplayName: "Smart — Opus 4.6, most capable", CompoundID: "amp-smart", ExtraFlags: []string{"-m", "smart"}, Recommended: true},
		{ID: "deep", DisplayName: "Deep — GPT-5.2 Codex, extended thinking", CompoundID: "amp-deep", ExtraFlags: []string{"-m", "deep"}},
	},
	"cursor-agent": {
		{ID: "opus-4.6-thinking", DisplayName: "Claude 4.6 Opus (Thinking) — default", CompoundID: "cursor-opus-4.6-thinking", ExtraFlags: []string{"--model", "opus-4.6-thinking"}, Recommended: true},
		{ID: "composer-1.5", DisplayName: "Composer 1.5", CompoundID: "cursor-composer-1.5", ExtraFlags: []string{"--model", "composer-1.5"}},
		{ID: "opus-4.6", DisplayName: "Claude 4.6 Opus", CompoundID: "cursor-opus-4.6", ExtraFlags: []string{"--model", "opus-4.6"}},
		{ID: "sonnet-4.6-thinking", DisplayName: "Claude 4.6 Sonnet (Thinking)", CompoundID: "cursor-sonnet-4.6-thinking", ExtraFlags: []string{"--model", "sonnet-4.6-thinking"}},
		{ID: "sonnet-4.6", DisplayName: "Claude 4.6 Sonnet", CompoundID: "cursor-sonnet-4.6", ExtraFlags: []string{"--model", "sonnet-4.6"}},
		{ID: "gpt-5.3-codex-xhigh-fast", DisplayName: "GPT-5.3 Codex Extra High Fast", CompoundID: "cursor-gpt-5.3-codex-xhigh-fast", ExtraFlags: []string{"--model", "gpt-5.3-codex-xhigh-fast"}},
		{ID: "gpt-5.3-codex-high", DisplayName: "GPT-5.3 Codex High", CompoundID: "cursor-gpt-5.3-codex-high", ExtraFlags: []string{"--model", "gpt-5.3-codex-high"}},
		{ID: "gemini-3-pro", DisplayName: "Gemini 3 Pro", CompoundID: "cursor-gemini-3-pro", ExtraFlags: []string{"--model", "gemini-3-pro"}},
		{ID: "gemini-3-flash", DisplayName: "Gemini 3 Flash", CompoundID: "cursor-gemini-3-flash", ExtraFlags: []string{"--model", "gemini-3-flash"}},
		{ID: "grok", DisplayName: "Grok", CompoundID: "cursor-grok", ExtraFlags: []string{"--model", "grok"}},
	},
}

// RecommendedModel returns the recommended model for an adapter, or nil if none exists.
func RecommendedModel(adapterName string) *Model {
	models, ok := AdapterModels[adapterName]
	if !ok {
		return nil
	}
	for i := range models {
		if models[i].Recommended {
			return &models[i]
		}
	}
	return nil
}
