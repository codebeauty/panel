package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

type ProgressModel struct {
	ToolIDs  []string
	Statuses map[string]*ToolProgress
	Spinner  spinner.Model
	Start    time.Time
}

func NewProgressModel(toolIDs []string) ProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	statuses := make(map[string]*ToolProgress, len(toolIDs))
	for _, id := range toolIDs {
		statuses[id] = &ToolProgress{Status: "pending"}
	}

	return ProgressModel{
		ToolIDs:  toolIDs,
		Statuses: statuses,
		Spinner:  s,
		Start:    time.Now(),
	}
}

func (m ProgressModel) AllDone() bool {
	for _, s := range m.Statuses {
		if s.Status == "pending" || s.Status == "running" {
			return false
		}
	}
	return true
}

func (m ProgressModel) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Running")
	elapsed := time.Since(m.Start).Round(time.Second)
	b.WriteString(fmt.Sprintf("  %s  %s\n\n", title, StyleMuted.Render(elapsed.String())))

	done := 0
	total := len(m.ToolIDs)

	for _, id := range m.ToolIDs {
		s := m.Statuses[id]
		switch s.Status {
		case "pending":
			b.WriteString(fmt.Sprintf("  %s %-20s %s\n",
				IconPending, id, StyleMuted.Render("waiting")))
		case "running":
			dur := time.Since(s.Started).Round(time.Second)
			b.WriteString(fmt.Sprintf("  %s %-20s %s  %s\n",
				m.Spinner.View(), id, StylePrimary.Render("running"), StyleMuted.Render(dur.String())))
		case "success":
			done++
			b.WriteString(fmt.Sprintf("  %s %-20s %s  %s  %s\n",
				IconSuccess, id, StyleSuccess.Render("done"),
				StyleMuted.Render(s.Duration.Round(time.Millisecond).String()),
				StyleMuted.Render(fmt.Sprintf("%d words", s.Words))))
		case "failed":
			done++
			b.WriteString(fmt.Sprintf("  %s %-20s %s  %s\n",
				IconError, id, StyleError.Render("failed"),
				StyleMuted.Render(s.Duration.Round(time.Millisecond).String())))
		case "timeout":
			done++
			b.WriteString(fmt.Sprintf("  %s %-20s %s  %s\n",
				StyleWarning.Render("⏱"), id, StyleWarning.Render("timeout"),
				StyleMuted.Render(s.Duration.Round(time.Millisecond).String())))
		case "cancelled":
			done++
			b.WriteString(fmt.Sprintf("  %s %-20s %s\n",
				StyleMuted.Render("−"), id, StyleMuted.Render("cancelled")))
		}
	}

	b.WriteString(fmt.Sprintf("\n  %s\n", StyleMuted.Render(fmt.Sprintf("%d/%d complete", done, total))))

	return b.String()
}
