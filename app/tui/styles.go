package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Premium Design System
var (
	// Colors (catppuccin mocha inspired)
	ColorRosewater = lipgloss.Color("#f5e0dc")
	ColorFlamingo  = lipgloss.Color("#f2cdcd")
	ColorPink      = lipgloss.Color("#f5c2e7")
	ColorMauve     = lipgloss.Color("#cba6f7")
	ColorRed       = lipgloss.Color("#f38ba8")
	ColorMaroon    = lipgloss.Color("#eba0ac")
	ColorPeach     = lipgloss.Color("#fab387")
	ColorYellow    = lipgloss.Color("#f9e2af")
	ColorGreen     = lipgloss.Color("#a6e3a1")
	ColorTeal      = lipgloss.Color("#94e2d5")
	ColorSky       = lipgloss.Color("#89dceb")
	ColorSapphire  = lipgloss.Color("#74c7ec")
	ColorBlue      = lipgloss.Color("#89b4fa")
	ColorLavender  = lipgloss.Color("#b4befe")
	ColorText      = lipgloss.Color("#cdd6f4")
	ColorSubtext1  = lipgloss.Color("#bac2de")
	ColorSubtext0  = lipgloss.Color("#a6adc8")
	ColorOverlay2  = lipgloss.Color("#9399b2")
	ColorOverlay1  = lipgloss.Color("#7f849c")
	ColorOverlay0  = lipgloss.Color("#6c7086")
	ColorSurface2  = lipgloss.Color("#585b70")
	ColorSurface1  = lipgloss.Color("#45475a")
	ColorSurface0  = lipgloss.Color("#313244")
	ColorBase      = lipgloss.Color("#1e1e2e")
	ColorMantle    = lipgloss.Color("#181825")
	ColorCrust     = lipgloss.Color("#11111b")

	// Styles
	StyleBase = lipgloss.NewStyle().Foreground(ColorText)

	StyleHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorMauve).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorLavender)

	StyleStatusConnected = lipgloss.NewStyle().
		Foreground(ColorGreen).
		SetString("● Connected")

	StyleStatusDisconnected = lipgloss.NewStyle().
		Foreground(ColorRed).
		SetString("○ Disconnected")

	StyleKey = lipgloss.NewStyle().
		Foreground(ColorPeach)

	StyleDesc = lipgloss.NewStyle().
		Foreground(ColorSubtext0)
)
