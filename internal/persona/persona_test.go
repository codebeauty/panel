package persona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/codebeauty/panel/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestBuiltinPresets(t *testing.T) {
	assert.Len(t, Builtins, 6)

	expected := []string{"security", "performance", "architect", "reviewer", "devil", "product"}
	for _, id := range expected {
		content, ok := Builtins[id]
		assert.True(t, ok, "missing builtin: %s", id)
		assert.NotEmpty(t, content, "empty builtin: %s", id)
	}
}

func TestBuiltinPresetsHaveRole(t *testing.T) {
	for id, content := range Builtins {
		assert.Contains(t, content, "You are", "builtin %s should define a role", id)
	}
}

func TestBuiltinIDs(t *testing.T) {
	ids := BuiltinIDs()
	assert.Len(t, ids, 6)
	// Should be sorted
	for i := 1; i < len(ids); i++ {
		assert.Less(t, ids[i-1], ids[i], "BuiltinIDs should be sorted")
	}
}

func TestValidatePersonaID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"security", true},
		{"my-expert", true},
		{"go_reviewer", true},
		{"expert.v2", true},
		{"../escape", false},
		{"path/traversal", false},
		{"has space", false},
		{"", false},
		{"semi;colon", false},
	}
	for _, tt := range tests {
		err := ValidatePersonaID(tt.id)
		if tt.valid {
			assert.NoError(t, err, "expected %q to be valid", tt.id)
		} else {
			assert.Error(t, err, "expected %q to be invalid", tt.id)
		}
	}
}

func TestPersonasDir(t *testing.T) {
	dir := PersonasDir()
	assert.Contains(t, dir, "panel")
	assert.True(t, strings.HasSuffix(dir, "personas"))
}

func TestPersonasDirMatchesConfigDir(t *testing.T) {
	personaDir := PersonasDir()
	configDir := config.GlobalConfigDir()
	assert.Equal(t, filepath.Join(configDir, "personas"), personaDir)
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	content := "You are a Go expert."
	os.WriteFile(filepath.Join(dir, "golang.md"), []byte(content), 0o600)

	loaded, err := Load("golang", dir)
	assert.NoError(t, err)
	assert.Equal(t, content, loaded)
}

func TestLoadNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := Load("nonexistent", dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestLoadValidatesID(t *testing.T) {
	dir := t.TempDir()
	_, err := Load("../escape", dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid persona ID")
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "security.md"), []byte("x"), 0o600)
	os.WriteFile(filepath.Join(dir, "custom.md"), []byte("y"), 0o600)
	os.WriteFile(filepath.Join(dir, "not-markdown.txt"), []byte("z"), 0o600)

	ids, err := List(dir)
	assert.NoError(t, err)
	assert.Equal(t, []string{"custom", "security"}, ids) // sorted, .md only
}

func TestListEmptyDir(t *testing.T) {
	dir := t.TempDir()
	ids, err := List(dir)
	assert.NoError(t, err)
	assert.Empty(t, ids)
}

func TestListMissingDir(t *testing.T) {
	ids, err := List("/nonexistent/path")
	assert.NoError(t, err)
	assert.Empty(t, ids)
}

func TestSyncBuiltinsNewDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "personas")
	written, err := SyncBuiltins(dir, nil)
	assert.NoError(t, err)
	assert.Equal(t, 6, written)

	ids, _ := List(dir)
	assert.Len(t, ids, 6)

	info, _ := os.Stat(filepath.Join(dir, "security.md"))
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestSyncBuiltinsSkipsIdentical(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "security.md"), []byte(Builtins["security"]), 0o600)

	written, err := SyncBuiltins(dir, nil)
	assert.NoError(t, err)
	assert.Equal(t, 5, written)
}

func TestSyncBuiltinsDiffCallback(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "security.md"), []byte("custom security persona"), 0o600)

	var diffCalled bool
	var diffID string
	callback := func(id, existing, builtin string) SyncAction {
		diffCalled = true
		diffID = id
		return SyncSkip
	}

	written, err := SyncBuiltins(dir, callback)
	assert.NoError(t, err)
	assert.True(t, diffCalled)
	assert.Equal(t, "security", diffID)
	assert.Equal(t, 5, written)

	data, _ := os.ReadFile(filepath.Join(dir, "security.md"))
	assert.Equal(t, "custom security persona", string(data))
}

func TestSyncBuiltinsOverwrite(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "security.md"), []byte("old content"), 0o600)

	callback := func(id, existing, builtin string) SyncAction {
		return SyncOverwrite
	}

	written, err := SyncBuiltins(dir, callback)
	assert.NoError(t, err)
	assert.Equal(t, 6, written)

	data, _ := os.ReadFile(filepath.Join(dir, "security.md"))
	assert.Equal(t, Builtins["security"], string(data))
}

func TestSyncBuiltinsBackup(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "security.md"), []byte("custom content"), 0o600)

	callback := func(id, existing, builtin string) SyncAction {
		return SyncBackup
	}

	written, err := SyncBuiltins(dir, callback)
	assert.NoError(t, err)
	assert.Equal(t, 6, written)

	data, _ := os.ReadFile(filepath.Join(dir, "security.md"))
	assert.Equal(t, Builtins["security"], string(data))

	backup, _ := os.ReadFile(filepath.Join(dir, "security.backup.md"))
	assert.Equal(t, "custom content", string(backup))
}

func TestSyncBuiltinsDeterministicOrder(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "architect.md"), []byte("custom"), 0o600)
	os.WriteFile(filepath.Join(dir, "security.md"), []byte("custom"), 0o600)

	var callOrder []string
	callback := func(id, existing, builtin string) SyncAction {
		callOrder = append(callOrder, id)
		return SyncSkip
	}

	SyncBuiltins(dir, callback)
	assert.Equal(t, []string{"architect", "security"}, callOrder)
}
