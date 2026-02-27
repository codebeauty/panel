package output

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)
var multiDash = regexp.MustCompile(`-{2,}`)

func Slug(prompt string) string {
	if prompt == "" {
		return "prompt"
	}
	s := strings.ToLower(prompt)
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = multiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
		s = strings.TrimRight(s, "-")
	}
	if s == "" {
		return "prompt"
	}
	return s
}

func RunDir(baseDir, prompt string) (string, error) {
	slug := Slug(prompt)
	ts := time.Now().Unix()
	dirName := fmt.Sprintf("%s-%d", slug, ts)
	path := filepath.Join(baseDir, dirName)
	if err := os.MkdirAll(path, 0o700); err != nil {
		return "", fmt.Errorf("creating output dir: %w", err)
	}
	return path, nil
}

func WritePrompt(dir, prompt string) error {
	return os.WriteFile(filepath.Join(dir, "prompt.md"), []byte(prompt), 0o600)
}

func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".horde-*")
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
