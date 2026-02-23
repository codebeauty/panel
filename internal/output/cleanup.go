package output

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Candidate represents an output directory eligible for cleanup.
type Candidate struct {
	Name  string
	Path  string
	Mtime time.Time
}

// ParseDuration parses a human-friendly duration string.
// Supports: ms, s, m, h, d, w. Bare number = days.
func ParseDuration(input string) (time.Duration, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Try suffixed formats
	suffixes := []struct {
		suffix string
		mult   time.Duration
	}{
		{"ms", time.Millisecond},
		{"s", time.Second},
		{"m", time.Minute},
		{"h", time.Hour},
		{"d", 24 * time.Hour},
		{"w", 7 * 24 * time.Hour},
	}

	for _, s := range suffixes {
		if strings.HasSuffix(input, s.suffix) {
			numStr := strings.TrimSuffix(input, s.suffix)
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid duration %q: %w", input, err)
			}
			return time.Duration(n * float64(s.mult)), nil
		}
	}

	// Bare number = days
	n, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", input, err)
	}
	return time.Duration(n * float64(24 * time.Hour)), nil
}

// scanDirs reads immediate subdirectories of baseDir, skipping files and symlinks.
func scanDirs(baseDir string) ([]Candidate, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading output dir: %w", err)
	}

	var dirs []Candidate
	for _, entry := range entries {
		// entry.Type() uses lstat, so ModeSymlink is visible here.
		// entry.IsDir() would return true for symlinks-to-dirs, so check type bits.
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		dirs = append(dirs, Candidate{
			Name:  entry.Name(),
			Path:  filepath.Join(baseDir, entry.Name()),
			Mtime: info.ModTime(),
		})
	}
	return dirs, nil
}

// ScanCandidates reads immediate children of baseDir,
// skips symlinks, filters directories by mtime before cutoff.
func ScanCandidates(baseDir string, cutoff time.Time) ([]Candidate, error) {
	dirs, err := scanDirs(baseDir)
	if err != nil {
		return nil, err
	}

	var candidates []Candidate
	for _, d := range dirs {
		if d.Mtime.Before(cutoff) {
			candidates = append(candidates, d)
		}
	}
	return candidates, nil
}

// ScanRuns returns all run directories in baseDir, sorted newest-first by mtime.
// Skips files and symlinks.
func ScanRuns(baseDir string) ([]Candidate, error) {
	runs, err := scanDirs(baseDir)
	if err != nil {
		return nil, err
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].Mtime.After(runs[j].Mtime)
	})
	return runs, nil
}
