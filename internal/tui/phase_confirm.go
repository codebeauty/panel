package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type ConfirmModel struct {
	ToolIDs []string
	Persona string
	Prompt  string
}

func (m ConfirmModel) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Confirm Dispatch")
	b.WriteString(fmt.Sprintf("  %s\n\n", title))

	// Prompt preview (truncated)
	prompt := m.Prompt
	if len(prompt) > 80 {
		prompt = prompt[:80] + "..."
	}
	prompt = strings.ReplaceAll(prompt, "\n", " ")
	b.WriteString(fmt.Sprintf("  %s  %s\n", StyleBold.Render("Prompt:"), prompt))

	// Tools
	b.WriteString(fmt.Sprintf("  %s   %s\n", StyleBold.Render("Tools:"), strings.Join(m.ToolIDs, ", ")))

	// Persona
	if m.Persona != "" {
		b.WriteString(fmt.Sprintf("  %s %s\n", StyleBold.Render("Persona:"), m.Persona))
	}

	b.WriteString(fmt.Sprintf("\n  %s\n", StyleMuted.Render("enter:dispatch  esc:back  q:quit")))

	return b.String()
}
