package style

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var White = lipgloss.Color("15")
var Red = lipgloss.Color("9")
var Green = lipgloss.Color("29")
var Yellow = lipgloss.Color("11")
var Purple = lipgloss.Color("99")
var Contrast = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffffff"}
var Highlight = lipgloss.Color("#30BA78")

// Badge renders the provided text as centered in a color background.
func Badge(text string, color lipgloss.TerminalColor) string {
	if len(text) > 12 {
		return lipgloss.NewStyle().Background(color).Width(12).Render(text)
	}
	return lipgloss.NewStyle().Background(color).Align(lipgloss.Center).Width(12).Render(text)
}

// FormatDuration returns a short string representation of a duration.
func FormatDuration(d time.Duration) string {
	totalMinutes := int(d.Minutes())
	days := totalMinutes / (24 * 60)
	hours := (totalMinutes % (24 * 60)) / 60
	minutes := totalMinutes % 60

	if days >= 365 {
		return fmt.Sprintf("%dy", days/365)
	}
	if days >= 2 {
		return fmt.Sprintf("%dd", days)
	}
	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	return fmt.Sprintf("%dh%dm", hours, minutes)
}
