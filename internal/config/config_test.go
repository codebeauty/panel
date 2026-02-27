package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobalConfigDir(t *testing.T) {
	dir := GlobalConfigDir()
	assert.True(t, strings.Contains(dir, "horde") || strings.Contains(dir, "panel"))
	assert.NotEmpty(t, dir)
}

func TestNewDefaults(t *testing.T) {
	cfg := NewDefaults()

	assert.Equal(t, 540, cfg.Defaults.Timeout)
	assert.Equal(t, "./agents/horde", cfg.Defaults.OutputDir)
	assert.Equal(t, ReadOnlyBestEffort, cfg.Defaults.ReadOnly)
	assert.Equal(t, 4, cfg.Defaults.MaxParallel)
	assert.NotNil(t, cfg.Tools)
	assert.NotNil(t, cfg.Groups)
}

func TestValidateToolName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"claude", true},
		{"claude-opus", true},
		{"my.tool", true},
		{"tool_1", true},
		{"../etc/passwd", false},
		{"tool name", false},
		{"tool;rm", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolName(tt.name)
			if tt.valid {
				assert.NoError(t, err, "expected %q to be valid", tt.name)
			} else {
				assert.Error(t, err, "expected %q to be invalid", tt.name)
			}
		})
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	data := `{
		"defaults": {"timeout": 300, "maxParallel": 2},
		"tools": {
			"claude": {
				"binary": "/usr/local/bin/claude",
				"adapter": "claude",
				"enabled": true
			}
		},
		"groups": {"fast": ["claude"]}
	}`
	os.WriteFile(cfgPath, []byte(data), 0o600)

	cfg, err := LoadFromFile(cfgPath)
	assert.NoError(t, err)
	assert.Equal(t, 300, cfg.Defaults.Timeout)
	assert.Equal(t, 2, cfg.Defaults.MaxParallel)
	assert.Equal(t, "./agents/horde", cfg.Defaults.OutputDir) // default preserved
	assert.Contains(t, cfg.Tools, "claude")
	assert.Equal(t, []string{"claude"}, cfg.Groups["fast"])
}

func TestLoadFromFileMissing(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.json")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := NewDefaults()
	cfg.Tools["claude"] = ToolConfig{
		Binary:  "/usr/local/bin/claude",
		Adapter: "claude",
		Enabled: true,
	}

	err := Save(cfg, path)
	assert.NoError(t, err)

	loaded, err := LoadFromFile(path)
	assert.NoError(t, err)
	assert.Equal(t, cfg.Defaults, loaded.Defaults)
	assert.Equal(t, cfg.Tools["claude"], loaded.Tools["claude"])

	// Verify permissions
	info, _ := os.Stat(path)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestLoadProjectConfig(t *testing.T) {
	// Test: file not found returns nil, nil
	pc, err := LoadProjectConfig("/nonexistent/path")
	assert.NoError(t, err)
	assert.Nil(t, pc)

	// Test: valid .horde.json
	dir := t.TempDir()
	timeout := 120
	outDir := "./custom/output"
	data := `{"defaults": {"timeout": 120, "outputDir": "./custom/output"}}`
	os.WriteFile(filepath.Join(dir, ".horde.json"), []byte(data), 0o600)

	pc, err = LoadProjectConfig(dir)
	assert.NoError(t, err)
	assert.NotNil(t, pc)
	assert.Equal(t, &timeout, pc.Defaults.Timeout)
	assert.Equal(t, &outDir, pc.Defaults.OutputDir)
}

func TestMergeWithProject(t *testing.T) {
	cfg := NewDefaults()

	timeout := 120
	ro := ReadOnlyEnforced
	pc := &ProjectConfig{
		Defaults: &ProjectDefaults{
			Timeout:  &timeout,
			ReadOnly: &ro,
		},
	}

	MergeWithProject(cfg, pc)
	assert.Equal(t, 120, cfg.Defaults.Timeout)
	assert.Equal(t, ReadOnlyEnforced, cfg.Defaults.ReadOnly) // enforced > bestEffort
	assert.Equal(t, "./agents/horde", cfg.Defaults.OutputDir) // unchanged
}

func TestMergeWithProjectNil(t *testing.T) {
	cfg := NewDefaults()
	MergeWithProject(cfg, nil) // should not panic
	assert.Equal(t, 540, cfg.Defaults.Timeout)
}

func TestMergeWithProjectReadOnlyClamp(t *testing.T) {
	cfg := NewDefaults() // defaults to bestEffort
	ro := ReadOnlyNone
	pc := &ProjectConfig{
		Defaults: &ProjectDefaults{
			ReadOnly: &ro,
		},
	}
	MergeWithProject(cfg, pc)
	// StricterReadOnly should keep bestEffort since it's stricter than none
	assert.Equal(t, ReadOnlyBestEffort, cfg.Defaults.ReadOnly)
}

func TestLoadMerged(t *testing.T) {
	dir := t.TempDir()
	data := `{"defaults": {"outputDir": "./custom/output"}}`
	os.WriteFile(filepath.Join(dir, ".horde.json"), []byte(data), 0o600)

	cfg, err := LoadMerged(dir)
	assert.NoError(t, err)
	assert.Equal(t, "./custom/output", cfg.Defaults.OutputDir)
	assert.Equal(t, 540, cfg.Defaults.Timeout)
}

func TestLoadMergedNoProjectConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadMerged(dir)
	assert.NoError(t, err)
	// Output dir comes from global config on disk (may be legacy ./agents/panel) or defaults (./agents/horde)
	assert.True(t, cfg.Defaults.OutputDir == "./agents/horde" || cfg.Defaults.OutputDir == "./agents/panel",
		"expected ./agents/horde or ./agents/panel, got %q", cfg.Defaults.OutputDir)
}

func TestToolConfigExpertField(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	data := `{
		"tools": {
			"claude": {
				"binary": "/usr/local/bin/claude",
				"adapter": "claude",
				"enabled": true,
				"expert": "security"
			}
		}
	}`
	os.WriteFile(cfgPath, []byte(data), 0o600)

	cfg, err := LoadFromFile(cfgPath)
	assert.NoError(t, err)
	assert.Equal(t, "security", cfg.Tools["claude"].Expert)
}

func TestToolConfigExpertOmitEmpty(t *testing.T) {
	cfg := NewDefaults()
	cfg.Tools["claude"] = ToolConfig{
		Binary:  "/usr/local/bin/claude",
		Adapter: "claude",
		Enabled: true,
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	err := Save(cfg, path)
	assert.NoError(t, err)

	data, _ := os.ReadFile(path)
	assert.NotContains(t, string(data), "expert")
}

func TestNewDefaultsHasTeams(t *testing.T) {
	cfg := NewDefaults()
	assert.NotNil(t, cfg.Teams)
	assert.Empty(t, cfg.Teams)
}

func TestLoadFromFileWithTeams(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{
		"tools": {"claude": {"binary": "claude", "adapter": "claude", "enabled": true}},
		"teams": {"code-review": ["security", "architect", "reviewer"]}
	}`
	os.WriteFile(cfgPath, []byte(data), 0o600)

	cfg, err := LoadFromFile(cfgPath)
	assert.NoError(t, err)
	assert.Equal(t, []string{"security", "architect", "reviewer"}, cfg.Teams["code-review"])
}

func TestLoadFromFileTeamsNilInit(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	data := `{"tools": {}}`
	os.WriteFile(cfgPath, []byte(data), 0o600)

	cfg, err := LoadFromFile(cfgPath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg.Teams)
}

func TestSaveAndLoadWithTeams(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := NewDefaults()
	cfg.Teams["review"] = []string{"security", "architect"}
	err := Save(cfg, path)
	assert.NoError(t, err)

	loaded, err := LoadFromFile(path)
	assert.NoError(t, err)
	assert.Equal(t, []string{"security", "architect"}, loaded.Teams["review"])
}

func TestStricterReadOnly(t *testing.T) {
	tests := []struct {
		a, b ReadOnlyMode
		want ReadOnlyMode
	}{
		{ReadOnlyNone, ReadOnlyEnforced, ReadOnlyEnforced},
		{ReadOnlyEnforced, ReadOnlyNone, ReadOnlyEnforced},
		{ReadOnlyBestEffort, ReadOnlyEnforced, ReadOnlyEnforced},
		{ReadOnlyBestEffort, ReadOnlyNone, ReadOnlyBestEffort},
		{ReadOnlyEnforced, ReadOnlyEnforced, ReadOnlyEnforced},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%s_vs_%s", tt.a, tt.b)
		t.Run(name, func(t *testing.T) {
			got := StricterReadOnly(tt.a, tt.b)
			assert.Equal(t, tt.want, got, "StricterReadOnly(%q, %q)", tt.a, tt.b)
		})
	}
}
