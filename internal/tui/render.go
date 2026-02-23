package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func Badge(text string) string {
	return StyleMuted.Render("(" + text + ")")
}

func StatusIcon(status string) string {
	switch status {
	case "success":
		return IconSuccess
	case "failed":
		return IconError
	case "timeout":
		return StyleWarning.Render("⏱")
	case "cancelled":
		return StyleMuted.Render("−")
	default:
		return StyleMuted.Render("?")
	}
}

func EnabledIcon(enabled bool) string {
	if enabled {
		return IconSuccess
	}
	return IconError
}

type Table struct {
	Headers []string
	Rows    [][]string
}

func (t Table) Render() string {
	if len(t.Headers) == 0 {
		return ""
	}

	// Calculate column widths (accounting for ANSI sequences in cells).
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = lipgloss.Width(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			w := lipgloss.Width(cell)
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	var b strings.Builder
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorMuted)

	// Header row
	b.WriteString("  ")
	for i, h := range t.Headers {
		cell := headerStyle.Render(h)
		b.WriteString(cell)
		if i < len(t.Headers)-1 {
			pad := widths[i] - lipgloss.Width(h) + 2
			b.WriteString(strings.Repeat(" ", pad))
		}
	}
	b.WriteString("\n")

	// Data rows
	for _, row := range t.Rows {
		b.WriteString("  ")
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			b.WriteString(cell)
			if i < len(row)-1 {
				pad := widths[i] - lipgloss.Width(cell) + 2
				if pad < 1 {
					pad = 1
				}
				b.WriteString(strings.Repeat(" ", pad))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

func FormatToolID(id string) string {
	if i := strings.Index(id, "@"); i >= 0 {
		return id[:i] + " " + StyleMuted.Render("("+id[i+1:]+")")
	}
	return id
}

func Separator(text string) string {
	line := StyleMuted.Render("───")
	return fmt.Sprintf("  %s %s %s", line, StyleBold.Render(text), line)
}
