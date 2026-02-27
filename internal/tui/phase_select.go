package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type SelectModel struct {
	items    []string
	selected map[int]bool
	cursor   int
	adapters map[string]string // toolID -> adapter name
}

func NewSelectModel(toolIDs []string, adapters map[string]string) SelectModel {
	sel := make(map[int]bool, len(toolIDs))
	for i := range toolIDs {
		sel[i] = true // all selected by default
	}
	return SelectModel{
		items:    toolIDs,
		selected: sel,
		adapters: adapters,
	}
}

func (m SelectModel) SelectedIDs() []string {
	var ids []string
	for i, id := range m.items {
		if m.selected[i] {
			ids = append(ids, id)
		}
	}
	return ids
}

func (m SelectModel) Update(msg tea.Msg) (SelectModel, tea.Cmd) {
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
		case key.Matches(msg, Keys.Toggle):
			m.selected[m.cursor] = !m.selected[m.cursor]
		case key.Matches(msg, Keys.All):
			for i := range m.items {
				m.selected[i] = true
			}
		case key.Matches(msg, Keys.None):
			for i := range m.items {
				m.selected[i] = false
			}
		}
	}
	return m, nil
}

func (m SelectModel) View() string {
	var b strings.Builder

	title := StyleTitle.Render("Select Agents")
	b.WriteString(fmt.Sprintf("  %s\n\n", title))

	for i, id := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = StylePrimary.Render("> ")
		}

		check := "[ ]"
		if m.selected[i] {
			check = StyleSuccess.Render("[✓]")
		}

		name := id
		if i == m.cursor {
			name = StyleBold.Render(id)
		}

		adapter := ""
		if a, ok := m.adapters[id]; ok {
			adapter = "  " + StyleMuted.Render(a)
		}

		b.WriteString(fmt.Sprintf("  %s%s %s%s\n", cursor, check, name, adapter))
	}

	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}

	b.WriteString(fmt.Sprintf("\n  %s selected", StyleMuted.Render(fmt.Sprintf("%d/%d", count, len(m.items)))))
	b.WriteString(fmt.Sprintf("  %s\n", StyleMuted.Render("space:toggle  a:all  n:none  enter:confirm  ctrl+c:quit")))

	return b.String()
}
