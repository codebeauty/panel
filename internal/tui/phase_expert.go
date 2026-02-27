package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type RaiderModel struct {
	items   []string // expert IDs, first entry is "(none)"
	cursor  int
	builtin map[string]bool
}

func NewRaiderModel(expertIDs []string, builtinSet map[string]bool) RaiderModel {
	items := append([]string{"(none)"}, expertIDs...)
	return RaiderModel{
		items:   items,
		builtin: builtinSet,
	}
}

func (m RaiderModel) SelectedExpert() string {
	if m.cursor == 0 {
		return ""
	}
	return m.items[m.cursor]
}

func (m RaiderModel) Update(msg tea.Msg) (RaiderModel, tea.Cmd) {
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

func (m RaiderModel) View() string {
	var b strings.Builder

	title := StyleTitle.Render("Select Raider")
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

	b.WriteString(fmt.Sprintf("\n  %s\n", StyleMuted.Render("↑/↓:navigate  enter:confirm  esc:back  ctrl+c:quit")))

	return b.String()
}
