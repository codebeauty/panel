package tui

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// IsTTY returns true when stdout is a terminal (not piped/redirected).
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Adaptive color palette — adjusts for light/dark terminals.
var (
	ColorSuccess = lipgloss.AdaptiveColor{Light: "#00A86B", Dark: "#73D16C"}
	ColorError   = lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF6B6B"}
	ColorWarning = lipgloss.AdaptiveColor{Light: "#FF8C00", Dark: "#FFAA44"}
	ColorPrimary = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	ColorMuted   = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#626262"}
)

// Shared text styles.
var (
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleError   = lipgloss.NewStyle().Foreground(ColorError)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)
	StylePrimary = lipgloss.NewStyle().Foreground(ColorPrimary)
	StyleMuted   = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleBold    = lipgloss.NewStyle().Bold(true)
)

// Status icons — colored Unicode symbols.
var (
	IconSuccess = StyleSuccess.Render("✓")
	IconError   = StyleError.Render("✗")
	IconWarning = StyleWarning.Render("⚠")
	IconPending = StyleMuted.Render("·")
)
