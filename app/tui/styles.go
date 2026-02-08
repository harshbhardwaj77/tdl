package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	// Colors (Mutable)
	ColorPrimary   lipgloss.Color
	ColorSecondary lipgloss.Color
	ColorError     lipgloss.Color
	ColorSuccess   lipgloss.Color
	ColorDim       lipgloss.Color

	// Text Styles
	TitleStyle        lipgloss.Style
	StatusBarStyle    lipgloss.Style

	// Item Styles
	SelectedItemStyle lipgloss.Style
	NormalItemStyle   lipgloss.Style

	// Pane Styles
	PaneStyle         lipgloss.Style
	ActivePaneStyle   lipgloss.Style
	InactivePaneStyle lipgloss.Style

	// Tab Styles
	TabStyle         lipgloss.Style
	ActiveTabStyle   lipgloss.Style
	InactiveTabStyle lipgloss.Style
)

func init() {
	// Default Theme
	InitStyles("62", "230", "196", "42", "240")
}

func InitStyles(primary, secondary, errorColor, success, dim string) {
	ColorPrimary   = lipgloss.Color(primary)
	ColorSecondary = lipgloss.Color(secondary)
	ColorError     = lipgloss.Color(errorColor)
	ColorSuccess   = lipgloss.Color(success)
	ColorDim       = lipgloss.Color(dim)

	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Padding(0, 1)

	SelectedItemStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(ColorPrimary).
			PaddingLeft(1)
	
	NormalItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	PaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder())

	ActivePaneStyle = PaneStyle.Copy().
			BorderForeground(ColorPrimary)

	InactivePaneStyle = PaneStyle.Copy().
			BorderForeground(ColorDim)

	TabStyle = lipgloss.NewStyle().
			Padding(0, 1)

	ActiveTabStyle = TabStyle.Copy().
			Foreground(ColorPrimary).
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorPrimary)

	InactiveTabStyle = TabStyle.Copy().
			Foreground(ColorDim).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorDim)
}

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
