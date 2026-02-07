package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorPrimary   = lipgloss.Color("62")  // Purple
	ColorSecondary = lipgloss.Color("230") // Light Cream
	ColorError     = lipgloss.Color("196") // Red
	ColorSuccess   = lipgloss.Color("42")  // Green
	ColorDim       = lipgloss.Color("240") // Grey

	// Text Styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Padding(0, 1)

	// Item Styles
	SelectedItemStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(ColorPrimary).
			PaddingLeft(1)
	
	NormalItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	// Pane Styles
	PaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder())

	ActivePaneStyle = PaneStyle.Copy().
			BorderForeground(ColorPrimary)

	InactivePaneStyle = PaneStyle.Copy().
			BorderForeground(ColorDim)
)

// Icons (Nerd Font friendly/Unicode)
const (
	IconFolder   = "📁"
	IconFile     = "📄"
	IconPhoto    = "🖼️"
	IconVideo    = "🎥"
	IconMusic    = "🎵"
	IconUnknown  = "❓"
	IconCheck    = "✅"
	IconError    = "❌"
	IconDownload = "⬇️"
	IconWaiting  = "⏳"
)
