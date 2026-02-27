package output

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"1d", 24 * time.Hour, false},
		{"2w", 336 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"500ms", 500 * time.Millisecond, false},
		{"7", 7 * 24 * time.Hour, false},
		{"", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScanCandidates(t *testing.T) {
	t.Run("filters by mtime", func(t *testing.T) {
		base := t.TempDir()

		oldDir := filepath.Join(base, "old-run")
		newDir := filepath.Join(base, "new-run")
		assert.NoError(t, os.Mkdir(oldDir, 0o700))
		assert.NoError(t, os.Mkdir(newDir, 0o700))

		// Set old dir mtime to 48 hours ago
		past := time.Now().Add(-48 * time.Hour)
		assert.NoError(t, os.Chtimes(oldDir, past, past))

		cutoff := time.Now().Add(-24 * time.Hour)
		candidates, err := ScanCandidates(base, cutoff)
		assert.NoError(t, err)
		assert.Len(t, candidates, 1)
		assert.Equal(t, "old-run", candidates[0].Name)
		assert.Equal(t, oldDir, candidates[0].Path)
	})

	t.Run("non-existent base dir returns nil", func(t *testing.T) {
		candidates, err := ScanCandidates("/tmp/horde-does-not-exist-"+t.Name(), time.Now())
		assert.NoError(t, err)
		assert.Nil(t, candidates)
	})
}

func TestScanRuns(t *testing.T) {
	t.Run("returns newest first", func(t *testing.T) {
		base := t.TempDir()

		dir1 := filepath.Join(base, "run-1")
		dir2 := filepath.Join(base, "run-2")
		dir3 := filepath.Join(base, "run-3")
		assert.NoError(t, os.Mkdir(dir1, 0o700))
		assert.NoError(t, os.Mkdir(dir2, 0o700))
		assert.NoError(t, os.Mkdir(dir3, 0o700))

		t2 := time.Now().Add(-48 * time.Hour)
		t3 := time.Now().Add(-24 * time.Hour)
		t1 := time.Now()
		assert.NoError(t, os.Chtimes(dir1, t1, t1))
		assert.NoError(t, os.Chtimes(dir2, t2, t2))
		assert.NoError(t, os.Chtimes(dir3, t3, t3))

		runs, err := ScanRuns(base)
		assert.NoError(t, err)
		assert.Len(t, runs, 3)
		assert.Equal(t, "run-1", runs[0].Name)
		assert.Equal(t, "run-3", runs[1].Name)
		assert.Equal(t, "run-2", runs[2].Name)
	})

	t.Run("skips symlinks to directories", func(t *testing.T) {
		base := t.TempDir()
		real := filepath.Join(base, "real-dir")
		assert.NoError(t, os.Mkdir(real, 0o700))
		assert.NoError(t, os.Symlink(real, filepath.Join(base, "link-dir")))

		runs, err := ScanRuns(base)
		assert.NoError(t, err)
		assert.Len(t, runs, 1)
		assert.Equal(t, "real-dir", runs[0].Name)
	})

	t.Run("skips files", func(t *testing.T) {
		base := t.TempDir()
		assert.NoError(t, os.Mkdir(filepath.Join(base, "real-dir"), 0o700))
		assert.NoError(t, os.WriteFile(filepath.Join(base, "a-file.txt"), []byte("hi"), 0o600))

		runs, err := ScanRuns(base)
		assert.NoError(t, err)
		assert.Len(t, runs, 1)
		assert.Equal(t, "real-dir", runs[0].Name)
	})

	t.Run("empty dir returns empty slice", func(t *testing.T) {
		base := t.TempDir()
		runs, err := ScanRuns(base)
		assert.NoError(t, err)
		assert.Empty(t, runs)
	})

	t.Run("non-existent dir returns nil", func(t *testing.T) {
		runs, err := ScanRuns("/tmp/horde-nonexistent-" + t.Name())
		assert.NoError(t, err)
		assert.Nil(t, runs)
	})
}
