package cli

import (
	"testing"

	"github.com/codebeauty/panel/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestExpandDuplicateToolIDs(t *testing.T) {
	cfg := config.NewDefaults()
	cfg.Tools["claude"] = config.ToolConfig{
		Binary:  "/usr/local/bin/claude",
		Adapter: "claude",
		Enabled: true,
	}

	ids := []string{"claude", "claude", "claude"}
	expanded := expandDuplicateToolIDs(ids, cfg)

	assert.Equal(t, []string{"claude", "claude__2", "claude__3"}, expanded)
	assert.Contains(t, cfg.Tools, "claude__2")
	assert.Contains(t, cfg.Tools, "claude__3")
	assert.Equal(t, cfg.Tools["claude"].Binary, cfg.Tools["claude__2"].Binary)
}

func TestExpandDuplicateToolIDsNoDuplicates(t *testing.T) {
	cfg := config.NewDefaults()
	cfg.Tools["claude"] = config.ToolConfig{Binary: "claude", Adapter: "claude", Enabled: true}
	cfg.Tools["gemini"] = config.ToolConfig{Binary: "gemini", Adapter: "gemini", Enabled: true}

	ids := []string{"claude", "gemini"}
	expanded := expandDuplicateToolIDs(ids, cfg)

	assert.Equal(t, []string{"claude", "gemini"}, expanded)
}
