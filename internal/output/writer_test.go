package output

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Review this authentication flow", "review-this-authentication-flow"},
		{"Hello, World! How are you?", "hello-world-how-are-you"},
		{"", "prompt"},
		{"a b  c---d", "a-b-c-d"},
	}
	for _, tt := range tests {
		got := Slug(tt.input)
		assert.Equal(t, tt.want, got, "Slug(%q)", tt.input)
		assert.LessOrEqual(t, len(got), 60)
	}
}

func TestWriteManifest(t *testing.T) {
	dir := t.TempDir()
	m := &Manifest{
		Version:     1,
		Prompt:      "test prompt",
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
		Duration:    "1s",
		Platform:    "darwin/arm64",
		Results:     []ManifestResult{},
	}
	err := WriteManifest(dir, m)
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "run.json"))
	assert.NoError(t, err)
	assert.Contains(t, string(data), "test prompt")
}
