package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type ReadOnlyMode string

const (
	ReadOnlyEnforced   ReadOnlyMode = "enforced"
	ReadOnlyBestEffort ReadOnlyMode = "bestEffort"
	ReadOnlyNone       ReadOnlyMode = "none"
)

type Config struct {
	Version  int                   `json:"version"`
	Defaults DefaultsConfig        `json:"defaults"`
	Tools    map[string]ToolConfig `json:"tools"`
	Groups   map[string][]string   `json:"groups"`
}

type DefaultsConfig struct {
	Timeout     int          `json:"timeout"`
	OutputDir   string       `json:"outputDir"`
	ReadOnly    ReadOnlyMode `json:"readOnly"`
	MaxParallel int          `json:"maxParallel"`
}

type ToolConfig struct {
	Binary     string   `json:"binary"`
	Adapter    string   `json:"adapter"`
	ExtraFlags []string `json:"extraFlags,omitempty"`
	Enabled    bool     `json:"enabled"`
	Stdin      bool     `json:"stdin,omitempty"`
}

func NewDefaults() *Config {
	return &Config{
		Version: 1,
		Defaults: DefaultsConfig{
			Timeout:     540,
			OutputDir:   "./agents/panel",
			ReadOnly:    ReadOnlyBestEffort,
			MaxParallel: 4,
		},
		Tools:  make(map[string]ToolConfig),
		Groups: make(map[string][]string),
	}
}

var validToolName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func ValidateToolName(name string) error {
	if !validToolName.MatchString(name) {
		return fmt.Errorf("invalid tool name %q: must match [a-zA-Z0-9._-]+", name)
	}
	return nil
}

func globalConfigDir() string {
	home := os.Getenv("HOME")
	macOSPath := filepath.Join(home, "Library", "Application Support", "panel")
	if _, err := os.Stat(macOSPath); err == nil {
		return macOSPath
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "panel")
	}
	return macOSPath
}

func GlobalConfigPath() string {
	return filepath.Join(globalConfigDir(), "config.json")
}

func LoadFromFile(path string) (*Config, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	// Security: refuse to load config writable by group/others
	if info.Mode().Perm()&0o022 != 0 {
		return nil, fmt.Errorf("config %s has unsafe permissions %o (writable by group/others)", path, info.Mode().Perm())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := NewDefaults()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.Tools == nil {
		cfg.Tools = make(map[string]ToolConfig)
	}
	if cfg.Groups == nil {
		cfg.Groups = make(map[string][]string)
	}
	return cfg, nil
}

func ValidateReadOnlyMode(mode string) (ReadOnlyMode, error) {
	switch ReadOnlyMode(mode) {
	case ReadOnlyEnforced, ReadOnlyBestEffort, ReadOnlyNone:
		return ReadOnlyMode(mode), nil
	case "":
		return ReadOnlyBestEffort, nil
	default:
		return "", fmt.Errorf("invalid read-only mode %q: must be enforced, bestEffort, or none", mode)
	}
}

var readOnlyStrictness = map[ReadOnlyMode]int{
	ReadOnlyNone:       0,
	ReadOnlyBestEffort: 1,
	ReadOnlyEnforced:   2,
}

func StricterReadOnly(a, b ReadOnlyMode) ReadOnlyMode {
	if readOnlyStrictness[a] >= readOnlyStrictness[b] {
		return a
	}
	return b
}

// ProjectDefaults holds per-project overrides loaded from .panel.json.
type ProjectDefaults struct {
	Timeout     *int          `json:"timeout,omitempty"`
	OutputDir   *string       `json:"outputDir,omitempty"`
	ReadOnly    *ReadOnlyMode `json:"readOnly,omitempty"`
	MaxParallel *int          `json:"maxParallel,omitempty"`
}

// ProjectConfig represents a .panel.json file in the project root.
type ProjectConfig struct {
	Defaults *ProjectDefaults `json:"defaults,omitempty"`
}

// LoadProjectConfig reads .panel.json from dir. Returns nil if not found.
func LoadProjectConfig(dir string) (*ProjectConfig, error) {
	path := filepath.Join(dir, ".panel.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading project config: %w", err)
	}
	var pc ProjectConfig
	if err := json.Unmarshal(data, &pc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &pc, nil
}

// MergeWithProject applies project-level overrides to the global config.
// Read-only is clamped via StricterReadOnly (project can only tighten, not loosen).
func MergeWithProject(cfg *Config, pc *ProjectConfig) {
	if pc == nil || pc.Defaults == nil {
		return
	}
	d := pc.Defaults
	if d.Timeout != nil {
		cfg.Defaults.Timeout = *d.Timeout
	}
	if d.OutputDir != nil {
		cfg.Defaults.OutputDir = *d.OutputDir
	}
	if d.ReadOnly != nil {
		cfg.Defaults.ReadOnly = StricterReadOnly(cfg.Defaults.ReadOnly, *d.ReadOnly)
	}
	if d.MaxParallel != nil {
		cfg.Defaults.MaxParallel = *d.MaxParallel
	}
}

func Load() (*Config, error) {
	path := GlobalConfigPath()
	cfg, err := LoadFromFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewDefaults(), nil
		}
		return nil, fmt.Errorf("global config: %w", err)
	}
	return cfg, nil
}

// LoadMerged loads global config, then merges project-level overrides from
// the .panel.json in the given directory. Use this for commands that need
// the resolved output directory (run, cleanup, summary).
func LoadMerged(projectDir string) (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	pc, err := LoadProjectConfig(projectDir)
	if err != nil {
		return nil, fmt.Errorf("project config: %w", err)
	}
	MergeWithProject(cfg, pc)
	return cfg, nil
}

func Save(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return atomicWrite(path, data, 0o600)
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".panel-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), perm); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
