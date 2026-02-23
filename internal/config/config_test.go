package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobalConfigDir(t *testing.T) {
	dir := GlobalConfigDir()
	assert.Contains(t, dir, "panel")
	assert.NotEmpty(t, dir)
}

func TestNewDefaults(t *testing.T) {
	cfg := NewDefaults()

	assert.Equal(t, 540, cfg.Defaults.Timeout)
	assert.Equal(t, "./agents/panel", cfg.Defaults.OutputDir)
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
		err := ValidateToolName(tt.name)
		if tt.valid {
			assert.NoError(t, err, "expected %q to be valid", tt.name)
		} else {
			assert.Error(t, err, "expected %q to be invalid", tt.name)
		}
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
	assert.Equal(t, "./agents/panel", cfg.Defaults.OutputDir) // default preserved
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

	// Test: valid .panel.json
	dir := t.TempDir()
	timeout := 120
	outDir := "./custom/output"
	data := `{"defaults": {"timeout": 120, "outputDir": "./custom/output"}}`
	os.WriteFile(filepath.Join(dir, ".panel.json"), []byte(data), 0o600)

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
	assert.Equal(t, "./agents/panel", cfg.Defaults.OutputDir) // unchanged
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
	os.WriteFile(filepath.Join(dir, ".panel.json"), []byte(data), 0o600)

	cfg, err := LoadMerged(dir)
	assert.NoError(t, err)
	assert.Equal(t, "./custom/output", cfg.Defaults.OutputDir)
	assert.Equal(t, 540, cfg.Defaults.Timeout)
}

func TestLoadMergedNoProjectConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadMerged(dir)
	assert.NoError(t, err)
	assert.Equal(t, "./agents/panel", cfg.Defaults.OutputDir)
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
		got := StricterReadOnly(tt.a, tt.b)
		assert.Equal(t, tt.want, got, "StricterReadOnly(%q, %q)", tt.a, tt.b)
	}
}
