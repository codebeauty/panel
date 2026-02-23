package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/codebeauty/panel/internal/adapter"
	"github.com/stretchr/testify/assert"
)

var mockBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "panel-test-*")
	if err != nil {
		panic(err)
	}
	mockBinary = filepath.Join(dir, "mocktool")

	src := filepath.Join(dir, "main.go")
	os.WriteFile(src, []byte(`package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: mocktool <stdout> <stderr> <exitcode> [delay_ms]")
		os.Exit(2)
	}
	if len(os.Args) > 4 {
		ms, _ := strconv.Atoi(os.Args[4])
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	fmt.Fprint(os.Stdout, os.Args[1])
	fmt.Fprint(os.Stderr, os.Args[2])
	code, _ := strconv.Atoi(os.Args[3])
	os.Exit(code)
}
`), 0o644)

	cmd := exec.Command("go", "build", "-o", mockBinary, src)
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(fmt.Sprintf("build mock: %v\n%s", err, out))
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type mockAdapter struct {
	name  string
	args  []string
	stdin string
}

func (m *mockAdapter) Name() string { return m.name }
func (m *mockAdapter) BuildInvocation(p adapter.RunParams) adapter.Invocation {
	return adapter.Invocation{
		Binary: mockBinary,
		Args:   m.args,
		Stdin:  m.stdin,
		Dir:    p.WorkDir,
	}
}
func (m *mockAdapter) ParseCost(stderr []byte) adapter.Cost { return adapter.Cost{} }

func TestRunnerSuccess(t *testing.T) {
	r := New(4)
	outDir := t.TempDir()

	tools := []Tool{
		{ID: "tool1", Adapter: &mockAdapter{name: "tool1", args: []string{"hello from tool1", "stderr1", "0"}}},
		{ID: "tool2", Adapter: &mockAdapter{name: "tool2", args: []string{"hello from tool2", "stderr2", "0"}}},
	}

	results := r.Run(context.Background(), tools, adapter.RunParams{
		Prompt:  "test",
		WorkDir: outDir,
		Timeout: 10 * time.Second,
	}, outDir)

	assert.Len(t, results, 2)
	assert.Equal(t, StatusSuccess, results[0].Status)
	assert.Equal(t, StatusSuccess, results[1].Status)
	assert.Equal(t, "hello from tool1", string(results[0].Stdout))
	assert.Equal(t, "hello from tool2", string(results[1].Stdout))
}

func TestRunnerTimeout(t *testing.T) {
	r := New(4)
	outDir := t.TempDir()

	tools := []Tool{
		{ID: "slow", Adapter: &mockAdapter{name: "slow", args: []string{"out", "err", "0", "5000"}}},
	}

	results := r.Run(context.Background(), tools, adapter.RunParams{
		Prompt:  "test",
		WorkDir: outDir,
		Timeout: 200 * time.Millisecond,
	}, outDir)

	assert.Len(t, results, 1)
	assert.Equal(t, StatusTimeout, results[0].Status)
}

func TestRunnerPartialFailure(t *testing.T) {
	r := New(4)
	outDir := t.TempDir()

	tools := []Tool{
		{ID: "good", Adapter: &mockAdapter{name: "good", args: []string{"ok", "", "0"}}},
		{ID: "bad", Adapter: &mockAdapter{name: "bad", args: []string{"", "error!", "1"}}},
	}

	results := r.Run(context.Background(), tools, adapter.RunParams{
		Prompt:  "test",
		WorkDir: outDir,
		Timeout: 10 * time.Second,
	}, outDir)

	assert.Len(t, results, 2)
	var good, bad Result
	for _, r := range results {
		if r.ToolID == "good" {
			good = r
		} else {
			bad = r
		}
	}
	assert.Equal(t, StatusSuccess, good.Status)
	assert.Equal(t, StatusFailed, bad.Status)
	assert.Equal(t, 1, bad.ExitCode)
}
