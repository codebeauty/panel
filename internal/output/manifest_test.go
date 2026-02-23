package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadManifest(t *testing.T) {
	t.Run("reads valid manifest", func(t *testing.T) {
		dir := t.TempDir()
		data := `{
			"version": 1,
			"prompt": "test prompt",
			"startedAt": "2026-02-23T00:06:34Z",
			"completedAt": "2026-02-23T00:07:01Z",
			"duration": "26.467s",
			"platform": "darwin/arm64",
			"config": {"readOnly": "bestEffort", "timeout": 540, "maxParallel": 4},
			"results": [
				{
					"toolId": "claude",
					"status": "success",
					"duration": "26.466s",
					"exitCode": 0,
					"outputFile": "claude.md",
					"stderrFile": "claude.stderr"
				}
			]
		}`
		err := os.WriteFile(filepath.Join(dir, "run.json"), []byte(data), 0o600)
		assert.NoError(t, err)

		m, err := ReadManifest(dir)
		assert.NoError(t, err)
		assert.Equal(t, "test prompt", m.Prompt)
		assert.Len(t, m.Results, 1)
		assert.Equal(t, "claude", m.Results[0].ToolID)
		assert.Equal(t, "success", m.Results[0].Status)
	})

	t.Run("missing run.json returns error", func(t *testing.T) {
		dir := t.TempDir()
		_, err := ReadManifest(dir)
		assert.Error(t, err)
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "run.json"), []byte("not json"), 0o600)
		assert.NoError(t, err)

		_, err = ReadManifest(dir)
		assert.Error(t, err)
	})
}

func TestManifestResultPersonaField(t *testing.T) {
	m := &Manifest{
		Results: []ManifestResult{
			{ToolID: "claude", Status: "success", Persona: "security"},
			{ToolID: "gemini", Status: "success"},
		},
	}

	data, err := json.Marshal(m)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"persona":"security"`)
}
