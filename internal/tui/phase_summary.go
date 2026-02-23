package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/codebeauty/panel/internal/runner"
)

type SummaryModel struct {
	Results []runner.Result
	RunDir  string
	Cursor  int
}

func NewSummaryModel(results []runner.Result, runDir string) SummaryModel {
	return SummaryModel{
		Results: results,
		RunDir:  runDir,
	}
}

func (m SummaryModel) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("Results")
	b.WriteString(fmt.Sprintf("  %s\n\n", title))

	var rows [][]string
	for _, r := range m.Results {
		icon := StatusIcon(string(r.Status))
		words := len(strings.Fields(string(r.Stdout)))
		dur := r.Duration.Round(time.Millisecond).String()
		rows = append(rows, []string{
			icon,
			r.ToolID,
			string(r.Status),
			StyleMuted.Render(dur),
			StyleMuted.Render(fmt.Sprintf("%d words", words)),
		})
	}

	t := Table{
		Headers: []string{"", "TOOL", "STATUS", "DURATION", "OUTPUT"},
		Rows:    rows,
	}
	b.WriteString(t.Render())

	b.WriteString(fmt.Sprintf("\n  %s %s\n", StyleBold.Render("Output:"), m.RunDir))
	b.WriteString(fmt.Sprintf("  %s\n", StyleMuted.Render("q:quit")))

	return b.String()
}
