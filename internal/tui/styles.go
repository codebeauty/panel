package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

var (
	ColorSuccess = lipgloss.AdaptiveColor{Light: "#00A86B", Dark: "#73D16C"}
	ColorError   = lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF6B6B"}
	ColorWarning = lipgloss.AdaptiveColor{Light: "#FF8C00", Dark: "#FFAA44"}
	ColorPrimary = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	ColorMuted   = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#626262"}
)

var (
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleError   = lipgloss.NewStyle().Foreground(ColorError)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)
	StylePrimary = lipgloss.NewStyle().Foreground(ColorPrimary)
	StyleMuted   = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleBold    = lipgloss.NewStyle().Bold(true)
	StyleTitle   = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
)

var (
	IconSuccess = StyleSuccess.Render("✓")
	IconError   = StyleError.Render("✗")
	IconWarning = StyleWarning.Render("⚠")
	IconPending = StyleMuted.Render("·")
)
