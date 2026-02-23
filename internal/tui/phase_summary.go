package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/codebeauty/panel/internal/runner"
)

const summaryChrome = 7 // title + blank + tab bar + separator + blank + footer output + footer keys

type SummaryModel struct {
	Results   []runner.Result
	RunDir    string
	activeTab int
	viewport  viewport.Model
	width     int
	height    int
	ready     bool
}

func NewSummaryModel(results []runner.Result, runDir string, width, height int) SummaryModel {
	m := SummaryModel{
		Results: results,
		RunDir:  runDir,
		width:   width,
		height:  height,
	}
	m.initViewport()
	return m
}

func (m *SummaryModel) initViewport() {
	vpHeight := m.height - summaryChrome
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport = viewport.New(m.viewportWidth(), vpHeight)
	m.viewport.SetContent(m.activeContent())
	m.ready = true
}

func (m *SummaryModel) activeContent() string {
	if len(m.Results) == 0 {
		return "(no results)"
	}
	r := m.Results[m.activeTab]
	content := strings.TrimSpace(string(r.Stdout))

	if content == "" && r.Status != "success" {
		content = m.failureInfo(r)
	} else if content == "" {
		content = StyleMuted.Render("(no output)")
	}

	return wordWrap(content, m.viewportWidth())
}

func (m *SummaryModel) failureInfo(r runner.Result) string {
	var b strings.Builder
	b.WriteString(StyleError.Render(fmt.Sprintf("Status: %s", r.Status)))
	if r.ExitCode != 0 {
		b.WriteString(fmt.Sprintf("  (exit code %d)", r.ExitCode))
	}
	b.WriteString("\n")

	// Try to diagnose the error and show a clear message first.
	if diag := runner.Diagnose(r.ToolID, r.Stderr, r.ExitCode); diag != nil {
		b.WriteString("\n")
		b.WriteString(StyleWarning.Render("⚑ " + diag.Message))
		b.WriteString("\n")
		b.WriteString(StyleMuted.Render("  " + diag.Suggestion))
		b.WriteString("\n")
	}

	stderr := strings.TrimSpace(string(r.Stderr))
	if stderr != "" {
		b.WriteString("\n")
		b.WriteString(StyleMuted.Render("stderr:"))
		b.WriteString("\n")
		b.WriteString(StyleMuted.Render(stderr))
	}
	return b.String()
}

func (m *SummaryModel) viewportWidth() int {
	w := m.width - 4
	if w < 20 {
		w = 20
	}
	return w
}

func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out strings.Builder
	for _, line := range strings.Split(s, "\n") {
		if lipgloss.Width(line) <= width {
			if out.Len() > 0 {
				out.WriteByte('\n')
			}
			out.WriteString(line)
			continue
		}
		words := strings.Fields(line)
		cur := 0
		for i, w := range words {
			ww := lipgloss.Width(w)
			if i == 0 {
				out.WriteString(w)
				cur = ww
				continue
			}
			if cur+1+ww > width {
				out.WriteByte('\n')
				out.WriteString(w)
				cur = ww
			} else {
				out.WriteByte(' ')
				out.WriteString(w)
				cur += 1 + ww
			}
		}
		if len(words) == 0 {
			if out.Len() > 0 {
				out.WriteByte('\n')
			}
		}
	}
	return out.String()
}

func (m *SummaryModel) switchTab(delta int) {
	if len(m.Results) == 0 {
		return
	}
	m.activeTab += delta
	if m.activeTab < 0 {
		m.activeTab = len(m.Results) - 1
	}
	if m.activeTab >= len(m.Results) {
		m.activeTab = 0
	}
	m.viewport.SetContent(m.activeContent())
	m.viewport.GotoTop()
}

func (m SummaryModel) Update(msg tea.Msg) (SummaryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Left):
			m.switchTab(-1)
			return m, nil
		case key.Matches(msg, Keys.Right):
			m.switchTab(1)
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.initViewport()
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m SummaryModel) View() string {
	var b strings.Builder

	title := StyleTitle.Render("Results")
	b.WriteString(fmt.Sprintf("  %s\n\n", title))

	// Tab bar
	b.WriteString("  ")
	for i, r := range m.Results {
		icon := StatusIcon(string(r.Status))
		name := FormatToolID(r.ToolID)

		var tab string
		if i == m.activeTab {
			tab = fmt.Sprintf("%s %s", icon, StyleBold.Render(name))
		} else {
			tab = fmt.Sprintf("%s %s", icon, StyleMuted.Render(name))
		}
		b.WriteString(tab)
		if i < len(m.Results)-1 {
			b.WriteString(StyleMuted.Render("  |  "))
		}
	}
	b.WriteString("\n")

	// Separator
	sep := strings.Repeat("─", m.viewportWidth())
	b.WriteString(fmt.Sprintf("  %s\n", StyleMuted.Render(sep)))

	// Viewport
	b.WriteString(m.indentViewport())

	// Footer
	b.WriteString(fmt.Sprintf("\n  %s %s\n", StyleBold.Render("Output:"), m.RunDir))
	scrollPct := fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100)
	hints := StyleMuted.Render(fmt.Sprintf("←/→:switch  ↑/↓:scroll  q:quit  %s", scrollPct))
	b.WriteString(fmt.Sprintf("  %s\n", hints))

	return b.String()
}

func (m SummaryModel) indentViewport() string {
	lines := strings.Split(m.viewport.View(), "\n")
	indent := lipgloss.NewStyle().PaddingLeft(2)
	for i, line := range lines {
		lines[i] = indent.Render(line)
	}
	return strings.Join(lines, "\n")
}
