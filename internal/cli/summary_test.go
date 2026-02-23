package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// setupRunDir creates a fake run directory with summary.md and run.json.
func setupRunDir(t *testing.T, base, name string, mtime time.Time) string {
	t.Helper()
	dir := filepath.Join(base, name)
	assert.NoError(t, os.MkdirAll(dir, 0o700))

	summary := "# Run Summary\n\n**Prompt:** " + name + "\n"
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "summary.md"), []byte(summary), 0o600))

	manifest := map[string]any{
		"version":     1,
		"prompt":      name,
		"startedAt":   mtime.Format(time.RFC3339),
		"completedAt": mtime.Add(30 * time.Second).Format(time.RFC3339),
		"duration":    "30s",
		"platform":    "darwin/arm64",
		"config":      map[string]any{"readOnly": "bestEffort", "timeout": 540, "maxParallel": 4},
		"results":     []any{},
	}
	data, _ := json.MarshalIndent(manifest, "", "  ")
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "run.json"), data, 0o600))

	assert.NoError(t, os.Chtimes(dir, mtime, mtime))
	return dir
}

func TestSummaryLatest(t *testing.T) {
	base := t.TempDir()
	now := time.Now()

	setupRunDir(t, base, "older-run-111", now.Add(-2*time.Hour))
	setupRunDir(t, base, "newer-run-222", now)

	t.Run("prints latest summary.md", func(t *testing.T) {
		var stdout bytes.Buffer
		root := newRootCmd()
		root.SetOut(&stdout)
		root.SetArgs([]string{"summary", "latest", "-o", base})
		err := root.Execute()
		assert.NoError(t, err)
		assert.Contains(t, stdout.String(), "newer-run-222")
		assert.NotContains(t, stdout.String(), "older-run-111")
	})

	t.Run("--path prints directory path", func(t *testing.T) {
		var stdout bytes.Buffer
		root := newRootCmd()
		root.SetOut(&stdout)
		root.SetArgs([]string{"summary", "latest", "-o", base, "--path"})
		err := root.Execute()
		assert.NoError(t, err)
		assert.Contains(t, stdout.String(), filepath.Join(base, "newer-run-222"))
	})

	t.Run("--json prints manifest", func(t *testing.T) {
		var stdout bytes.Buffer
		root := newRootCmd()
		root.SetOut(&stdout)
		root.SetArgs([]string{"summary", "latest", "-o", base, "--json"})
		err := root.Execute()
		assert.NoError(t, err)
		var m map[string]any
		assert.NoError(t, json.Unmarshal(stdout.Bytes(), &m))
		assert.Equal(t, "newer-run-222", m["prompt"])
	})

	t.Run("empty dir returns error", func(t *testing.T) {
		empty := t.TempDir()
		root := newRootCmd()
		root.SetArgs([]string{"summary", "latest", "-o", empty})
		root.SilenceErrors = true
		root.SilenceUsage = true
		err := root.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no runs found")
	})
}

func TestSummaryList(t *testing.T) {
	base := t.TempDir()
	now := time.Now()

	setupRunDir(t, base, "run-a-111", now.Add(-3*time.Hour))
	setupRunDir(t, base, "run-b-222", now.Add(-1*time.Hour))
	setupRunDir(t, base, "run-c-333", now)

	t.Run("lists runs as cards", func(t *testing.T) {
		var stdout bytes.Buffer
		root := newRootCmd()
		root.SetOut(&stdout)
		root.SetArgs([]string{"summary", "list", "-o", base})
		err := root.Execute()
		assert.NoError(t, err)
		out := stdout.String()
		assert.Contains(t, out, "run-c-333")
		assert.Contains(t, out, "run-b-222")
		assert.Contains(t, out, "run-a-111")
		// Newest should appear before oldest
		cPos := bytes.Index([]byte(out), []byte("run-c-333"))
		aPos := bytes.Index([]byte(out), []byte("run-a-111"))
		assert.Less(t, cPos, aPos)
	})

	t.Run("respects --limit", func(t *testing.T) {
		var stdout bytes.Buffer
		root := newRootCmd()
		root.SetOut(&stdout)
		root.SetArgs([]string{"summary", "list", "-o", base, "--limit", "2"})
		err := root.Execute()
		assert.NoError(t, err)
		out := stdout.String()
		assert.Contains(t, out, "run-c-333")
		assert.Contains(t, out, "run-b-222")
		assert.NotContains(t, out, "run-a-111")
	})

	t.Run("--json outputs array", func(t *testing.T) {
		var stdout bytes.Buffer
		root := newRootCmd()
		root.SetOut(&stdout)
		root.SetArgs([]string{"summary", "list", "-o", base, "--json"})
		err := root.Execute()
		assert.NoError(t, err)
		var arr []map[string]any
		assert.NoError(t, json.Unmarshal(stdout.Bytes(), &arr))
		assert.Len(t, arr, 3)
		assert.Equal(t, "run-c-333", arr[0]["prompt"])
	})

	t.Run("empty dir", func(t *testing.T) {
		empty := t.TempDir()
		var stdout bytes.Buffer
		root := newRootCmd()
		root.SetOut(&stdout)
		root.SetErr(&bytes.Buffer{})
		root.SetArgs([]string{"summary", "list", "-o", empty})
		err := root.Execute()
		assert.NoError(t, err)
	})
}
