package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case loginMsg:
		if msg.Err != nil {
			m.Connected = false
		} else {
			m.Connected = true
			m.User = msg.User
		}
	case ProgressMsg:
		item, exists := m.Downloads[msg.Name]
		if !exists {
			// New download
			prog := progress.New(progress.WithDefaultGradient())
			item = &DownloadItem{
				Name:     msg.Name,
				Total:    msg.Total,
				Progress: prog,
			}
			m.Downloads[msg.Name] = item
		}
		
		if msg.Err != nil {
			item.Err = msg.Err
			item.Finished = true
		} else if msg.IsFinished {
			item.Finished = true
			item.Downloaded = item.Total
		} else {
			item.Downloaded = msg.State.Downloaded
			item.Total = msg.State.Total // Update total just in case
		}
		
		// Update progress bar model
		// Calculate percentage
		// Update progress bar model
		// Calculate percentage
		// var pct float64
		// if item.Total > 0 {
		// 	pct = float64(item.Downloaded) / float64(item.Total)
		// }
		// We don't really have a cmd from progress update usually unless it animates
		// But here we just set percentage for view
		// Actually bubbles/progress needs an update msg for animation, but we can just View() it with strict percentage if we want
		// or use SetPercent
		
		// For now simple reliable approach:
		// We are not using the bubble's internal ticking for smooth animation to keep it simple first
		
		return m, nil
	case tea.KeyMsg:
		// If input is focused, pass messages to it
		if m.input.Focused() {
			switch msg.String() {
			case "enter":
				url := m.input.Value()
				m.input.Reset()
				m.input.Blur()
				return m, m.startDownload(url)
			case "esc":
				m.input.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "d":
			m.state = stateDashboard
		case "l":
			m.state = stateDownloads
		case "i":
			m.input.Focus()
			return m, textinput.Blink
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}
