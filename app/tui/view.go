package tui

import (
	"fmt"
	"strings"
	"sort"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if m.quitting {
		return "Bye!\n"
	}

	var s string

	// Header
	header := TitleStyle.Render("TDL TUI")
	status := lipgloss.NewStyle().Foreground(ColorError).Render("Disconnected")
	if m.Connected {
		status = lipgloss.NewStyle().Foreground(ColorSuccess).Render("Connected")
	}
	
	s += lipgloss.JoinHorizontal(lipgloss.Center, header, "  ", status)
	s += "\n\n"

	// Main Content
	// Handle different tabs
	switch m.state {
	case stateConfig:
		s += m.viewConfig()
	case stateBatch:
		s += m.viewBatch()
	case stateDownloads:
		s += m.viewDownloads()
	default:
		// ActiveTab handling when on dashboard/browser
		if m.ActiveTab == 1 {
			s += m.viewBrowser()
		} else if m.ActiveTab == 2 {
			s += m.viewDownloads()
		} else {
			s += m.viewDashboard()
		}
	}

	return s
}

func (m *Model) viewBrowser() string {
	var s string
	
	// Left Pane (Dialogs)
	leftStyle := InactivePaneStyle.Copy().
		Width(m.width / 3).
		Height(m.height - 4)
		
	if m.Pane == 0 {
		leftStyle = ActivePaneStyle.Copy().
			Width(m.width / 3).
			Height(m.height - 4)
	}
	
	// Left Content
	var leftContent string
	if m.LoadingDialogs {
		leftContent = fmt.Sprintf("\n\n   %s Loading chats...", m.spinner.View())
	} else {
		leftContent = m.Dialogs.View()
	}
	left := leftStyle.Render(leftContent)
	
	// Right Pane (Messages)
	rightStyle := InactivePaneStyle.Copy().
		Width((m.width / 3) * 2).
		Height(m.height - 4).
		MarginLeft(1)

	if m.Pane == 1 {
		rightStyle = ActivePaneStyle.Copy().
			Width((m.width / 3) * 2).
			Height(m.height - 4).
			MarginLeft(1)
	}
	
	// Right Content
	var rightContent string
	if m.LoadingHistory {
		rightContent = fmt.Sprintf("\n\n   %s Loading messages...", m.spinner.View())
	} else if len(m.Messages.Items()) > 0 {
		rightContent = m.Messages.View()
	} else {
		rightContent = "Select a chat to view messages"
	}
	right := rightStyle.Render(rightContent)
	
	s = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	s += StatusBarStyle.Render("\n  [Tab] Switch Pane • [Enter] Select • [e] Export Info (JSON) • [Esc] Back")
	
	if m.LoadingExport {
		s += lipgloss.NewStyle().Foreground(ColorPrimary).Render("\n  ⏳ Exporting chat info... This may take a while.")
	} else if m.StatusMessage != "" {
		s += lipgloss.NewStyle().Foreground(ColorSuccess).Render("\n  " + m.StatusMessage)
	}
	
	return s
}

func (m *Model) viewDashboard() string {
	var s strings.Builder

	s.WriteString("Welcome to TDL TUI\n\n")

	if m.Connected {
		s.WriteString(lipgloss.NewStyle().Foreground(ColorSuccess).Render("  You are connected to Telegram."))
		if m.User != nil {
			user := fmt.Sprintf("\n  User: %s %s (@%s)\n  ID: %d", 
				m.User.FirstName, m.User.LastName, m.User.Username, m.User.ID)
			s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(user))
		}
	} else {
		s.WriteString(lipgloss.NewStyle().Foreground(ColorError).Render("  Not connected."))
		s.WriteString("\n  Please login via 'tdl login' first or check your configuration.\n")
	}
	
	s.WriteString("\n\n  [d] Dashboard  [b] Browser  [l] Downloads  [i] New Download  [q] Quit")
	
	if m.input.Focused() {
		s.WriteString("\n\n")
		s.WriteString(m.input.View())
	}
	
	// Footer
	s.WriteString("\n\n")
	s.WriteString(StatusBarStyle.Render(fmt.Sprintf("tdl %s • %s", m.BuildInfo, m.Namespace)))
	
	return s.String()
}

func (m *Model) viewDownloads() string {
	var s strings.Builder
	s.WriteString("Active Downloads:\n\n")
	
	if len(m.Downloads) == 0 {
		s.WriteString("  No active downloads.\n")
		return s.String()
	}
	
	// Sort by name for stability
	keys := make([]string, 0, len(m.Downloads))
	for k := range m.Downloads {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	for _, k := range keys {
		item := m.Downloads[k]
		
		// Bar
		var pct float64
		if item.Total > 0 {
			pct = float64(item.Downloaded) / float64(item.Total)
		}
		
		// Update bar view manually without message loop for now
		// In a real app we'd trigger updates
		
		bar := item.Progress.ViewAs(pct)
		
		status := fmt.Sprintf("%s  %s", item.Name, bar)
		if item.Finished {
			if item.Err != nil {
				status += lipgloss.NewStyle().Foreground(ColorError).Render(" Failed")
			} else {
				status += lipgloss.NewStyle().Foreground(ColorSuccess).Render(" Done")
			}
		}
		
		s.WriteString("  " + status + "\n")
	}
	
	return s.String()
}

func (m *Model) viewBatch() string {
	var s strings.Builder
	s.WriteString(TitleStyle.Render("Batch Download (JSON)"))
	s.WriteString("\n\n  Select a JSON file containing message/media data:\n\n")
	s.WriteString(m.FilePicker.View() + "\n")
	s.WriteString(StatusBarStyle.Render("\n  [Esc] Back • [Enter] Select Directory/File"))
	return s.String()
}

func (m *Model) helpView() string {
	return StatusBarStyle.Render("\n  ctrl+c/q: quit • d: dashboard • l: download • b: browser • c: config • j: batch (json)\n")
}
