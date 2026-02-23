package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

func padToolID(id string, maxWidth int) string {
	display := FormatToolID(id)
	pad := max(0, maxWidth-lipgloss.Width(display))
	return display + strings.Repeat(" ", pad)
}

type ProgressModel struct {
	ToolIDs    []string
	Statuses   map[string]*ToolProgress
	Spinner    spinner.Model
	Start      time.Time
	maxIDWidth int // max visual width of formatted tool IDs
}

func NewProgressModel(toolIDs []string) ProgressModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = StylePrimary

	statuses := make(map[string]*ToolProgress, len(toolIDs))
	maxW := 0
	for _, id := range toolIDs {
		statuses[id] = &ToolProgress{Status: "pending"}
		if w := lipgloss.Width(FormatToolID(id)); w > maxW {
			maxW = w
		}
	}

	return ProgressModel{
		ToolIDs:    toolIDs,
		Statuses:   statuses,
		Spinner:    s,
		Start:      time.Now(),
		maxIDWidth: maxW,
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

const statusWidth = 9 // visual width of the longest status label ("cancelled")

func padStatus(styled, label string) string {
	pad := max(0, statusWidth-len(label))
	return styled + strings.Repeat(" ", pad)
}

func (m ProgressModel) View() string {
	var b strings.Builder

	title := StyleTitle.Render("Running")
	elapsed := time.Since(m.Start).Round(time.Second)
	b.WriteString(fmt.Sprintf("  %s  %s\n\n", title, StyleMuted.Render(elapsed.String())))

	// Pre-compute max duration width for alignment.
	maxDurW := 0
	for _, id := range m.ToolIDs {
		s := m.Statuses[id]
		var durStr string
		switch s.Status {
		case "running":
			durStr = time.Since(s.Started).Round(time.Second).String()
		case "pending", "cancelled":
			// no duration column
		default:
			durStr = s.Duration.Round(time.Millisecond).String()
		}
		if w := len(durStr); w > maxDurW {
			maxDurW = w
		}
	}

	done := 0
	for _, id := range m.ToolIDs {
		s := m.Statuses[id]
		pid := padToolID(id, m.maxIDWidth)

		switch s.Status {
		case "pending":
			b.WriteString(fmt.Sprintf("  %s %s %s\n",
				IconPending, pid, padStatus(StyleMuted.Render("waiting"), "waiting")))
		case "running":
			durStr := time.Since(s.Started).Round(time.Second).String()
			dur := StyleMuted.Render(fmt.Sprintf("%-*s", maxDurW, durStr))
			b.WriteString(fmt.Sprintf("  %s %s %s %s\n",
				m.Spinner.View(), pid, padStatus(StylePrimary.Render("running"), "running"), dur))
		case "success":
			done++
			durStr := fmt.Sprintf("%-*s", maxDurW, s.Duration.Round(time.Millisecond).String())
			dur := StyleMuted.Render(durStr)
			words := StyleMuted.Render(fmt.Sprintf("%d words", s.Words))
			b.WriteString(fmt.Sprintf("  %s %s %s %s  %s\n",
				IconSuccess, pid, padStatus(StyleSuccess.Render("done"), "done"), dur, words))
		case "failed":
			done++
			durStr := fmt.Sprintf("%-*s", maxDurW, s.Duration.Round(time.Millisecond).String())
			dur := StyleMuted.Render(durStr)
			b.WriteString(fmt.Sprintf("  %s %s %s %s\n",
				IconError, pid, padStatus(StyleError.Render("failed"), "failed"), dur))
		case "timeout":
			done++
			durStr := fmt.Sprintf("%-*s", maxDurW, s.Duration.Round(time.Millisecond).String())
			dur := StyleMuted.Render(durStr)
			b.WriteString(fmt.Sprintf("  %s %s %s %s\n",
				StyleWarning.Render("⏱"), pid, padStatus(StyleWarning.Render("timeout"), "timeout"), dur))
		case "cancelled":
			done++
			b.WriteString(fmt.Sprintf("  %s %s %s\n",
				StyleMuted.Render("−"), pid, padStatus(StyleMuted.Render("cancelled"), "cancelled")))
		}
	}

	b.WriteString(fmt.Sprintf("\n  %s\n",
		StyleMuted.Render(fmt.Sprintf("%d/%d complete", done, len(m.ToolIDs)))))

	return b.String()
}
