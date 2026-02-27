package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/codebeauty/horde/internal/runner"
)

func noopDispatch(_ context.Context, _ []string, _ string) {}

func TestPhaseTransition_SelectToExpertToConfirm(t *testing.T) {
	cfg := RunConfig{
		AllToolIDs: []string{"claude", "gemini"},
		Adapters:   map[string]string{"claude": "claude", "gemini": "gemini"},
		ExpertIDs:  []string{"security"},
		BuiltinSet: map[string]bool{"security": true},
		Prompt:     "test prompt",
	}
	m := NewModel(cfg, noopDispatch)
	assert.Equal(t, PhaseSelect, m.phase)

	// Confirm in select phase (all selected by default)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)
	assert.Equal(t, PhaseRaider, m.phase)

	// Confirm in expert phase
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)
	assert.Equal(t, PhaseConfirm, m.phase)
	assert.True(t, m.confirmModel.CanGoBack)
}

func TestPhaseTransition_SkipSelectSkipExpert_GoesToConfirm(t *testing.T) {
	cfg := RunConfig{
		AllToolIDs: []string{"claude", "gemini"},
		Adapters:   map[string]string{"claude": "claude", "gemini": "gemini"},
		Prompt:     "test",
		SkipSelect: true,
		SkipExpert: true,
		PreExpert:  "security",
	}
	m := NewModel(cfg, noopDispatch)
	// Multiple tools with expert -> confirm phase
	assert.Equal(t, PhaseConfirm, m.phase)
	assert.False(t, m.confirmModel.CanGoBack)
}

func TestPhaseTransition_SkipSelectSkipExpert_SingleTool_GoesToProgress(t *testing.T) {
	cfg := RunConfig{
		AllToolIDs: []string{"claude"},
		Adapters:   map[string]string{"claude": "claude"},
		Prompt:     "test",
		SkipSelect: true,
		SkipExpert: true,
	}
	m := NewModel(cfg, noopDispatch)
	assert.Equal(t, PhaseProgress, m.phase)
}

func TestPhaseTransition_SkipSelect_GoesToExpert(t *testing.T) {
	cfg := RunConfig{
		AllToolIDs: []string{"claude"},
		Adapters:   map[string]string{"claude": "claude"},
		ExpertIDs:  []string{"security"},
		BuiltinSet: map[string]bool{"security": true},
		Prompt:     "test",
		SkipSelect: true,
	}
	m := NewModel(cfg, noopDispatch)
	assert.Equal(t, PhaseRaider, m.phase)
}

func TestBackNavigation_ExpertToSelect(t *testing.T) {
	cfg := RunConfig{
		AllToolIDs: []string{"claude"},
		Adapters:   map[string]string{"claude": "claude"},
		ExpertIDs:  []string{"security"},
		BuiltinSet: map[string]bool{"security": true},
		Prompt:     "test",
	}
	m := NewModel(cfg, noopDispatch)
	assert.Equal(t, PhaseSelect, m.phase)

	// Go to expert
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)
	assert.Equal(t, PhaseRaider, m.phase)

	// Back to select
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = result.(Model)
	assert.Equal(t, PhaseSelect, m.phase)
}

func TestBackNavigation_ConfirmToExpert(t *testing.T) {
	cfg := RunConfig{
		AllToolIDs: []string{"claude"},
		Adapters:   map[string]string{"claude": "claude"},
		ExpertIDs:  []string{"security"},
		BuiltinSet: map[string]bool{"security": true},
		Prompt:     "test",
	}
	m := NewModel(cfg, noopDispatch)

	// Select -> Expert -> Confirm
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)
	assert.Equal(t, PhaseConfirm, m.phase)

	// Back to expert
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = result.(Model)
	assert.Equal(t, PhaseRaider, m.phase)
}

func TestBackNavigation_ConfirmStuck_WhenBothSkipped(t *testing.T) {
	cfg := RunConfig{
		AllToolIDs: []string{"claude", "gemini"},
		Adapters:   map[string]string{"claude": "claude", "gemini": "gemini"},
		Prompt:     "test",
		SkipSelect: true,
		SkipExpert: true,
		PreExpert:  "security",
	}
	m := NewModel(cfg, noopDispatch)
	assert.Equal(t, PhaseConfirm, m.phase)
	assert.False(t, m.confirmModel.CanGoBack)

	// Esc does nothing (stays on confirm)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = result.(Model)
	assert.Equal(t, PhaseConfirm, m.phase)
}

func TestQuitOnlyCtrlC_InSelectPhase(t *testing.T) {
	cfg := RunConfig{
		AllToolIDs: []string{"claude"},
		Adapters:   map[string]string{"claude": "claude"},
		Prompt:     "test",
	}
	m := NewModel(cfg, noopDispatch)

	// 'q' should NOT quit in select phase (it's been removed from Quit binding)
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = result.(Model)
	assert.False(t, m.quitting)
	assert.Nil(t, cmd)
}

func TestQuitQ_InSummaryPhase(t *testing.T) {
	cfg := RunConfig{
		AllToolIDs: []string{"claude"},
		Adapters:   map[string]string{"claude": "claude"},
		Prompt:     "test",
		SkipSelect: true,
		SkipExpert: true,
	}
	m := NewModel(cfg, noopDispatch)
	// Force into summary phase
	m.phase = PhaseSummary
	m.summaryModel = SummaryModel{}

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = result.(Model)
	assert.True(t, m.quitting)
	assert.NotNil(t, cmd) // tea.Quit
}

func TestConfirmToProgress_SetsCancel(t *testing.T) {
	dispatched := false
	dispatch := func(_ context.Context, _ []string, _ string) {
		dispatched = true
	}
	cfg := RunConfig{
		AllToolIDs: []string{"claude", "gemini"},
		Adapters:   map[string]string{"claude": "claude", "gemini": "gemini"},
		Prompt:     "test",
		SkipSelect: true,
		SkipExpert: true,
		PreExpert:  "",
	}
	m := NewModel(cfg, dispatch)
	assert.Equal(t, PhaseConfirm, m.phase)

	// Confirm -> Progress
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(Model)
	assert.Equal(t, PhaseProgress, m.phase)
	assert.NotNil(t, m.cancel, "cancel should be set in Update, not in Cmd")

	// Verify cancel works
	m.cancel()
	_ = dispatched // dispatch is called async via Cmd, not synchronously
}

func TestFormatToolID(t *testing.T) {
	// Plain ID
	assert.Equal(t, "claude", FormatToolID("claude"))

	// Composite ID — contains styled text, check visual content
	formatted := FormatToolID("claude@security")
	assert.Contains(t, formatted, "claude")
	assert.Contains(t, formatted, "security")
}

func testResults() []runner.Result {
	return []runner.Result{
		{ToolID: "claude", Status: "success", Stdout: []byte("claude output"), Duration: time.Second},
		{ToolID: "gemini", Status: "success", Stdout: []byte("gemini output"), Duration: 2 * time.Second},
		{ToolID: "codex", Status: "failed", Stdout: []byte("codex output"), Duration: 3 * time.Second},
	}
}

func TestSummaryModel_TabSwitching(t *testing.T) {
	sm := NewSummaryModel(testResults(), "/tmp/run", 80, 24)
	assert.Equal(t, 0, sm.activeTab)

	// Right arrow -> next tab
	sm, _ = sm.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, sm.activeTab)

	// Right again
	sm, _ = sm.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 2, sm.activeTab)

	// Right wraps to first
	sm, _ = sm.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 0, sm.activeTab)

	// Left wraps to last
	sm, _ = sm.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 2, sm.activeTab)
}

func TestSummaryModel_ViewContainsTabBar(t *testing.T) {
	sm := NewSummaryModel(testResults(), "/tmp/run", 80, 24)
	view := sm.View()

	assert.Contains(t, view, "claude")
	assert.Contains(t, view, "gemini")
	assert.Contains(t, view, "codex")
	assert.Contains(t, view, "Results")
	assert.Contains(t, view, "Output:")
	assert.Contains(t, view, "←/→:switch")
}

func TestSummaryModel_ViewShowsActiveContent(t *testing.T) {
	sm := NewSummaryModel(testResults(), "/tmp/run", 80, 24)
	view := sm.View()
	assert.Contains(t, view, "claude output")

	// Switch to gemini tab
	sm, _ = sm.Update(tea.KeyMsg{Type: tea.KeyRight})
	view = sm.View()
	assert.Contains(t, view, "gemini output")
}

func TestSummaryModel_EmptyResults(t *testing.T) {
	sm := NewSummaryModel(nil, "/tmp/run", 80, 24)
	view := sm.View()
	assert.Contains(t, view, "no results")
}

func TestSummaryModel_EmptyStdout_Success(t *testing.T) {
	results := []runner.Result{
		{ToolID: "claude", Status: "success", Stdout: []byte(""), Duration: time.Second},
	}
	sm := NewSummaryModel(results, "/tmp/run", 80, 24)
	view := sm.View()
	assert.Contains(t, view, "no output")
}

func TestSummaryModel_FailedTool_ShowsErrorInfo(t *testing.T) {
	results := []runner.Result{
		{ToolID: "gemini", Status: "failed", Stdout: []byte(""), Stderr: []byte("connection refused"), ExitCode: 1, Duration: time.Second},
	}
	sm := NewSummaryModel(results, "/tmp/run", 80, 24)
	view := sm.View()
	assert.Contains(t, view, "failed")
	assert.Contains(t, view, "exit code 1")
	assert.Contains(t, view, "connection refused")
	assert.Contains(t, view, "stderr")
}

func TestSummaryModel_FailedTool_NoStderr(t *testing.T) {
	results := []runner.Result{
		{ToolID: "gemini", Status: "timeout", Stdout: []byte(""), Duration: time.Second},
	}
	sm := NewSummaryModel(results, "/tmp/run", 80, 24)
	view := sm.View()
	assert.Contains(t, view, "timeout")
	assert.NotContains(t, view, "stderr")
}

func TestSummaryModel_WordWrap(t *testing.T) {
	long := strings.Repeat("word ", 30) // 150 chars, should wrap at width 80
	results := []runner.Result{
		{ToolID: "claude", Status: "success", Stdout: []byte(long), Duration: time.Second},
	}
	sm := NewSummaryModel(results, "/tmp/run", 80, 24)
	view := sm.View()
	// Content should be present and wrapped (multiple lines)
	lines := strings.Split(view, "\n")
	assert.Greater(t, len(lines), 5) // chrome + at least 2 content lines
}

func TestSummaryModel_WindowResize(t *testing.T) {
	sm := NewSummaryModel(testResults(), "/tmp/run", 80, 24)
	sm, _ = sm.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	assert.Equal(t, 120, sm.width)
	assert.Equal(t, 40, sm.height)
}
