package gather

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGatherFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello world"), 0o600)
	os.WriteFile(filepath.Join(dir, "big.txt"), []byte(strings.Repeat("x", 200*1024)), 0o600)

	result, err := Gather([]string{"hello.txt"}, 50, dir)
	assert.NoError(t, err)
	assert.Contains(t, result, "hello world")
	assert.Contains(t, result, "#### hello.txt")
}

func TestGatherFilesOverBudget(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "big.txt"), []byte(strings.Repeat("x", 200*1024)), 0o600)

	result, err := Gather([]string{"big.txt"}, 50, dir)
	assert.NoError(t, err)
	assert.Empty(t, result) // file too large, skipped; no git repo so no diff
}

func TestGatherMissingFile(t *testing.T) {
	dir := t.TempDir()
	result, err := Gather([]string{"nonexistent.txt"}, 50, dir)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestBuildPrompt(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		context string
		wantAll []string
	}{
		{
			"with context",
			"How should I refactor this?",
			"some code here",
			[]string{"## Question", "How should I refactor this?", "## Context", "some code here", "## Instructions"},
		},
		{
			"without context",
			"What is the best approach?",
			"",
			[]string{"## Question", "What is the best approach?", "## Instructions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildPrompt(tt.prompt, tt.context)
			for _, want := range tt.wantAll {
				assert.Contains(t, got, want)
			}
			if tt.context == "" {
				assert.NotContains(t, got, "## Context")
			}
		})
	}
}
