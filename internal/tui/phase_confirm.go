package tui

import (
	"fmt"
	"strings"
)

type ConfirmModel struct {
	ToolIDs   []string
	Expert    string
	Prompt    string
	CanGoBack bool
}

func (m ConfirmModel) View() string {
	var b strings.Builder

	title := StyleTitle.Render("Confirm Deploy")
	b.WriteString(fmt.Sprintf("  %s\n\n", title))

	// Prompt preview (truncated)
	prompt := m.Prompt
	if len(prompt) > 80 {
		prompt = prompt[:80] + "..."
	}
	prompt = strings.ReplaceAll(prompt, "\n", " ")
	b.WriteString(fmt.Sprintf("  %s  %s\n", StyleBold.Render("Prompt:"), prompt))

	// Tools
	var displayIDs []string
	for _, id := range m.ToolIDs {
		displayIDs = append(displayIDs, FormatToolID(id))
	}
	b.WriteString(fmt.Sprintf("  %s   %s\n", StyleBold.Render("Tools:"), strings.Join(displayIDs, ", ")))

	// Raider
	if m.Expert != "" {
		b.WriteString(fmt.Sprintf("  %s  %s\n", StyleBold.Render("Raider:"), m.Expert))
	}

	hints := "enter:deploy"
	if m.CanGoBack {
		hints += "  esc:back"
	}
	hints += "  ctrl+c:quit"
	b.WriteString(fmt.Sprintf("\n  %s\n", StyleMuted.Render(hints)))

	return b.String()
}
