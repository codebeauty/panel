package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/codebeauty/panel/internal/adapter"
)

var allowedEnvKeys = map[string]bool{
	"PATH": true, "HOME": true, "USER": true, "SHELL": true, "TERM": true,
	"ANTHROPIC_API_KEY": true, "OPENAI_API_KEY": true,
	"GEMINI_API_KEY": true, "GOOGLE_API_KEY": true,
	"AMP_API_KEY": true,
	"HTTP_PROXY": true, "HTTPS_PROXY": true, "NO_PROXY": true,
	"TMPDIR": true,
}

var injectedEnv = []string{
	"CI=true",
	"NO_COLOR=1",
}

const maxOutputBytes = 10 * 1024 * 1024 // 10MB per stream

type Tool struct {
	ID      string
	Adapter adapter.Adapter
}

type ProgressFunc func(toolID, event string, result *Result)

type Runner struct {
	maxParallel int64
	onProgress  ProgressFunc
}

func New(maxParallel int) *Runner {
	if maxParallel < 1 {
		maxParallel = 4
	}
	return &Runner{maxParallel: int64(maxParallel)}
}

func (r *Runner) SetProgressFunc(fn ProgressFunc) {
	r.onProgress = fn
}

func FilterEnv() []string {
	var filtered []string
	for _, env := range os.Environ() {
		if i := strings.IndexByte(env, '='); i > 0 && allowedEnvKeys[env[:i]] {
			filtered = append(filtered, env)
		}
	}
	return filtered
}

func (r *Runner) Run(ctx context.Context, tools []Tool, params adapter.RunParams, outDir string) []Result {
	results := make([]Result, len(tools))

	sem := semaphore.NewWeighted(r.maxParallel)
	g, gctx := errgroup.WithContext(ctx)

	if params.Env == nil {
		params.Env = append(FilterEnv(), injectedEnv...)
	}

	for i, tool := range tools {
		g.Go(func() error {
			if err := sem.Acquire(gctx, 1); err != nil {
				results[i] = Result{ToolID: tool.ID, Status: StatusCancelled}
				return nil
			}
			defer sem.Release(1)

			results[i] = r.execTool(gctx, tool, params, outDir)
			return nil
		})
	}

	g.Wait()
	return results
}

// RunWithParams dispatches each tool with its own RunParams in parallel.
// The params slice must be the same length as tools.
func (r *Runner) RunWithParams(ctx context.Context, tools []Tool, params []adapter.RunParams, outDir string) []Result {
	results := make([]Result, len(tools))

	sem := semaphore.NewWeighted(r.maxParallel)
	g, gctx := errgroup.WithContext(ctx)

	env := append(FilterEnv(), injectedEnv...)

	for i, tool := range tools {
		p := params[i]
		if p.Env == nil {
			p.Env = env
		}
		g.Go(func() error {
			if err := sem.Acquire(gctx, 1); err != nil {
				results[i] = Result{ToolID: tool.ID, Status: StatusCancelled}
				return nil
			}
			defer sem.Release(1)

			results[i] = r.execTool(gctx, tool, p, outDir)
			return nil
		})
	}

	g.Wait()
	return results
}

func (r *Runner) execTool(ctx context.Context, tool Tool, params adapter.RunParams, outDir string) Result {
	start := time.Now()

	toolCtx, cancel := context.WithTimeout(ctx, params.Timeout)
	defer cancel()

	inv := tool.Adapter.BuildInvocation(params)

	cmd := exec.CommandContext(toolCtx, inv.Binary, inv.Args...)
	cmd.Dir = inv.Dir
	cmd.Env = params.Env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdoutBuf, stderrBuf bytes.Buffer

	stdoutPath := filepath.Join(outDir, tool.ID+".md")
	stdoutFile, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return Result{
			ToolID:   tool.ID,
			Status:   StatusFailed,
			Duration: time.Since(start),
			Stderr:   []byte(fmt.Sprintf("failed to create output file: %v", err)),
		}
	}
	defer stdoutFile.Close()

	cmd.Stdout = io.MultiWriter(stdoutFile, &limitedWriter{buf: &stdoutBuf, max: maxOutputBytes})
	cmd.Stderr = &limitedWriter{buf: &stderrBuf, max: maxOutputBytes}

	if inv.Stdin != "" {
		cmd.Stdin = strings.NewReader(inv.Stdin)
	}

	if err := cmd.Start(); err != nil {
		return Result{
			ToolID:   tool.ID,
			Status:   StatusFailed,
			Duration: time.Since(start),
			Stderr:   []byte(err.Error()),
			ExitCode: -1,
		}
	}

	if r.onProgress != nil {
		r.onProgress(tool.ID, "started", nil)
	}

	waitErr := cmd.Wait()
	duration := time.Since(start)

	stderrPath := filepath.Join(outDir, tool.ID+".stderr")
	if err := os.WriteFile(stderrPath, stderrBuf.Bytes(), 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write stderr for %s: %v\n", tool.ID, err)
	}

	result := Result{
		ToolID:   tool.ID,
		Stdout:   stripANSI(stdoutBuf.Bytes()),
		Stderr:   stripANSI(stderrBuf.Bytes()),
		Duration: duration,
		Cost:     tool.Adapter.ParseCost(stderrBuf.Bytes()),
	}

	if waitErr != nil {
		ctxErr := toolCtx.Err()
		switch {
		case errors.Is(ctxErr, context.DeadlineExceeded):
			result.Status = StatusTimeout
		case errors.Is(ctxErr, context.Canceled):
			result.Status = StatusCancelled
		default:
			result.Status = StatusFailed
		}
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		r.killProcessGroup(cmd)
	} else {
		result.Status = StatusSuccess
		result.ExitCode = 0
	}

	if r.onProgress != nil {
		r.onProgress(tool.ID, "completed", &result)
	}

	return result
}

func (r *Runner) killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	syscall.Kill(-pid, syscall.SIGTERM)
	time.AfterFunc(5*time.Second, func() {
		syscall.Kill(-pid, syscall.SIGKILL)
	})
}

type limitedWriter struct {
	buf       *bytes.Buffer
	max       int
	truncated bool
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.truncated {
		return len(p), nil
	}
	if w.buf.Len()+len(p) > w.max {
		remaining := w.max - w.buf.Len()
		if remaining > 0 {
			w.buf.Write(p[:remaining])
		}
		w.buf.WriteString("\n[output truncated at 10MB]\n")
		w.truncated = true
		return len(p), nil
	}
	return w.buf.Write(p)
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(b []byte) []byte {
	return ansiPattern.ReplaceAll(b, nil)
}
