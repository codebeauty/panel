package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/codebeauty/panel/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestExpandTeamCrossProduct(t *testing.T) {
	cfg := config.NewDefaults()
	cfg.Tools["claude"] = config.ToolConfig{Binary: "claude", Adapter: "claude", Enabled: true}
	cfg.Tools["gemini"] = config.ToolConfig{Binary: "gemini", Adapter: "gemini", Enabled: true}

	result := expandTeamCrossProduct([]string{"claude", "gemini"}, []string{"security", "architect"}, cfg)

	assert.Equal(t, []string{
		"claude@security", "claude@architect",
		"gemini@security", "gemini@architect",
	}, result)

	// Composite tools registered in config
	assert.Contains(t, cfg.Tools, "claude@security")
	assert.Contains(t, cfg.Tools, "gemini@architect")
	assert.Equal(t, "claude", cfg.Tools["claude@security"].Binary)
	assert.Equal(t, "gemini", cfg.Tools["gemini@architect"].Binary)
}

func TestExpandTeamDeduplicatesTools(t *testing.T) {
	cfg := config.NewDefaults()
	cfg.Tools["claude"] = config.ToolConfig{Binary: "claude", Adapter: "claude", Enabled: true}

	result := expandTeamCrossProduct([]string{"claude", "claude"}, []string{"security"}, cfg)

	assert.Equal(t, []string{"claude@security"}, result)
}

func TestExpandTeamSingleToolSingleExpert(t *testing.T) {
	cfg := config.NewDefaults()
	cfg.Tools["claude"] = config.ToolConfig{Binary: "claude", Adapter: "claude", Enabled: true}

	result := expandTeamCrossProduct([]string{"claude"}, []string{"security"}, cfg)

	assert.Equal(t, []string{"claude@security"}, result)
}

func TestResolveTeamExperts(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "security.md"), []byte("You are a security expert."), 0o600)
	os.WriteFile(filepath.Join(dir, "architect.md"), []byte("You are an architect."), 0o600)

	ids, contents, err := resolveTeamExperts(
		[]string{"claude@security", "gemini@architect", "claude@architect"},
		dir,
	)
	assert.NoError(t, err)
	assert.Equal(t, []string{"security", "architect", "architect"}, ids)
	assert.Equal(t, "You are a security expert.", contents[0])
	assert.Equal(t, "You are an architect.", contents[1])
	assert.Equal(t, "You are an architect.", contents[2]) // cached
}

func TestResolveTeamExpertsMissing(t *testing.T) {
	dir := t.TempDir()

	_, _, err := resolveTeamExperts([]string{"claude@nonexistent"}, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestResolveTeamExpertsNoAtSign(t *testing.T) {
	dir := t.TempDir()

	ids, contents, err := resolveTeamExperts([]string{"plain-tool"}, dir)
	assert.NoError(t, err)
	assert.Equal(t, "", ids[0])
	assert.Equal(t, "", contents[0])
}
