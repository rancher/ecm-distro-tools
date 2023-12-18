package main

import "github.com/charmbracelet/lipgloss"

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
