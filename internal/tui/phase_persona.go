package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PersonaModel struct {
	items   []string // persona IDs, first entry is "(none)"
	cursor  int
	builtin map[string]bool
}

func NewPersonaModel(personaIDs []string, builtinSet map[string]bool) PersonaModel {
	items := append([]string{"(none)"}, personaIDs...)
	return PersonaModel{
		items:   items,
		builtin: builtinSet,
	}
}

func (m PersonaModel) SelectedPersona() string {
	if m.cursor == 0 {
		return ""
	}
	return m.items[m.cursor]
}

func (m PersonaModel) Update(msg tea.Msg) (PersonaModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, Keys.Down):
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m PersonaModel) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Select Persona")
	b.WriteString(fmt.Sprintf("  %s\n\n", title))

	for i, id := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = StylePrimary.Render("> ")
		}

		radio := StyleMuted.Render("○")
		if i == m.cursor {
			radio = StylePrimary.Render("●")
		}

		name := id
		if i == m.cursor {
			name = StyleBold.Render(id)
		}

		badge := ""
		if m.builtin[id] {
			badge = "  " + Badge("built-in")
		}

		b.WriteString(fmt.Sprintf("  %s%s %s%s\n", cursor, radio, name, badge))
	}

	b.WriteString(fmt.Sprintf("\n  %s\n", StyleMuted.Render("↑/↓:navigate  enter:confirm  esc:back  q:quit")))

	return b.String()
}
