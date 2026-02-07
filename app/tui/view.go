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
	header := StyleHeader.Render("TDL TUI")
	status := StyleStatusDisconnected.String()
	if m.Connected {
		status = StyleStatusConnected.String()
	}
	
	s += lipgloss.JoinHorizontal(lipgloss.Center, header, "  ", status)
	s += "\n\n"

	// Main Content
	switch m.state {
	case stateDashboard:
		s += m.viewDashboard()
	case stateDownloads:
		s += m.viewDownloads()
	}

	// Footer
	s += "\n\n"
	s += StyleDesc.Render(fmt.Sprintf("tdl %s • %s", m.BuildInfo, m.Namespace))
	s += "\n"
	s += m.helpView()

	return s
}

func (m *Model) viewDashboard() string {
	var s strings.Builder

	s.WriteString("Welcome to TDL TUI\n\n")

	if m.Connected {
		s.WriteString(StyleStatusConnected.Render("  You are connected to Telegram."))
		if m.User != nil {
			user := fmt.Sprintf("\n  User: %s %s (@%s)\n  ID: %d", 
				m.User.FirstName, m.User.LastName, m.User.Username, m.User.ID)
			s.WriteString(lipgloss.NewStyle().Foreground(ColorText).Render(user))
		}
	} else {
		s.WriteString(StyleStatusDisconnected.Render("  Not connected."))
		s.WriteString("\n  Please login via 'tdl login' first or check your configuration.\n")
	}
	
	s.WriteString("\n\n  [d] Dashboard  [l] Downloads  [i] New Download  [q] Quit")
	
	if m.input.Focused() {
		s.WriteString("\n\n")
		s.WriteString(m.input.View())
	}
	
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
				status += StyleStatusDisconnected.Render(" Failed")
			} else {
				status += StyleStatusConnected.Render(" Done")
			}
		}
		
		s.WriteString("  " + status + "\n")
	}
	
	return s.String()
}

func (m *Model) helpView() string {
	return StyleDesc.Render("\n  ctrl+c/q: quit • d: dashboard • l: downloads\n")
}
