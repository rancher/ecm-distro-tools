package main

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var selected = lipgloss.Color("32")
var docStyle = lipgloss.NewStyle().Margin(1, 2)
var dot = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(" • ")
var subtle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
var greenText = lipgloss.NewStyle().Foreground(lipgloss.Color("29"))
var redText = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
var yellowText = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
var yellowBlock = lipgloss.NewStyle().Background(lipgloss.Color("11")).Padding(0, 1)
var greenBlock = lipgloss.NewStyle().Background(lipgloss.Color("29")).Padding(0, 1)
var redBlock = lipgloss.NewStyle().Background(lipgloss.Color("9")).Padding(0, 1)
var purpleBlock = lipgloss.NewStyle().Background(lipgloss.Color("99")).Padding(0, 1)
var greyBlock = lipgloss.NewStyle().Background(lipgloss.Color("241")).Padding(0, 1)
var whiteBlock = lipgloss.NewStyle().Background(lipgloss.Color("15")).Padding(0, 1)

func formatDuration(d time.Duration) string {
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
