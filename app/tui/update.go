package tui

import (
	"fmt"
	"os"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/gotd/td/tg"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global Config Editor Intercept
	if m.state == stateConfig {
		return m.updateConfig(msg)
	}

	// Batch File Picker Intercept
	if m.state == stateBatch {
		var cmd tea.Cmd
		m.FilePicker, cmd = m.FilePicker.Update(msg)
		
		if didSelect, path := m.FilePicker.DidSelectFile(msg); didSelect {
			// Trigger download with file
			m.state = stateDownloads
			m.ActiveTab = 2 // Switch to downloads tab
			return m, m.startBatchDownload(path)
		}
		
		// Handle Esc to exit
		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" {
			m.state = stateDashboard
		}
		
		return m, cmd
	}

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
	
	case dialogsMsg:
		m.LoadingDialogs = false
		if msg.Err != nil {
			// Handle error (maybe show in status bar)
		} else {
			items := make([]list.Item, len(msg.Dialogs))
			for i, d := range msg.Dialogs {
				items[i] = d
			}
			m.Dialogs.SetItems(items)
		}
	
	case historyMsg:
		m.LoadingHistory = false
		if msg.Err != nil {
			// Handle error
		} else {
			items := make([]list.Item, len(msg.Messages))
			for i, m := range msg.Messages {
				items[i] = m
			}
			m.Messages.SetItems(items)
		}

	case ExportMsg:
		m.LoadingExport = false
		if msg.Err != nil {
			m.StatusMessage = fmt.Sprintf("Export Failed: %v", msg.Err)
		} else {
			m.StatusMessage = fmt.Sprintf("Exported to %s", msg.Path)
		}

		
		if msg.Err != nil {
			// handle global or item specific error
		}
		
		// Fallthrough only if we have logic for item updates here
		// But in this case we seem to have mixed logic from ProgressMsg.
		// The original ProgressMsg block handles 'item'.
		
		return m, nil
		
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
		// Global Navigation
		switch msg.String() {
		case "tab":
			if m.ActiveTab == 1 {
				m.Pane = 1 - m.Pane // Toggle 0/1
				return m, nil
			}
		}

		// If input is focused, pass messages to it
		if m.input.Focused() {
			switch msg.String() {
			case "enter":
				url := m.input.Value()
				m.input.Reset()
				m.input.Blur()
				m.ActiveTab = 2 // Switch to downloads
				return m, m.startDownload(url)
			case "esc":
				m.input.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		// List handling if in Browser
		if m.ActiveTab == 1 {
			var cmd tea.Cmd
			if m.Pane == 0 {
				m.Dialogs, cmd = m.Dialogs.Update(msg)
				// Handle Enter on Dialog
				if msg.String() == "enter" {
					// Fetch history for selected dialog
					if dlg, ok := m.Dialogs.SelectedItem().(DialogItem); ok {
						m.Messages.SetItems(nil) // Clear previous
						m.LoadingHistory = true
						return m, tea.Batch(m.GetHistory(dlg.Peer), m.spinner.Tick)
					}
					m.Pane = 1
				}
				// Handle Export
				if msg.String() == "e" {
					if dlg, ok := m.Dialogs.SelectedItem().(DialogItem); ok {
						m.LoadingExport = true
						return m, tea.Batch(m.startExport(dlg), m.spinner.Tick)
					}
				}
				return m, cmd
			} else {
				m.Messages, cmd = m.Messages.Update(msg)
				if msg.String() == "enter" {
					if mItem, ok := m.Messages.SelectedItem().(MessageItem); ok {
						if mItem.HasMedia || mItem.Text != "" {
							// Construct URL
							// Format: https://t.me/c/CHANNEL_ID/MSG_ID
							// Works for private groups/channels if user is member
							// For public: t.me/USERNAME/MSG_ID (we don't have username handy easily, but c/ID works for members)
							
							// Note: ChatID is int64. t.me links use ID without -100 prefix for supergroups usually?
							// Actually tdl supports internal format or standard t.me links.
							// For private chats (users), this might fail.
							// Let's try best effort for channels/chats.
							
							var link string
							// check peer type
							switch mItem.Peer.(type) {
							case *tg.InputPeerUser:
								// Not supported easily via URL yet without heavy lifting
								// Maybe specific "user" handler needed
							default:
								link = fmt.Sprintf("https://t.me/c/%d/%d", mItem.ChatID, mItem.ID)
							}
							
							if link != "" {
								m.ActiveTab = 2 // Switch to downloads
								return m, m.startDownload(link)
							}
						}
					}
				}
				return m, cmd
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "d":
			m.state = stateDashboard
			m.ActiveTab = 0
		case "b": // Browser
			m.ActiveTab = 1
			// Trigger fetch dialogs if empty
			if len(m.Dialogs.Items()) == 0 {
				m.LoadingDialogs = true
				return m, tea.Batch(m.GetDialogs(), m.spinner.Tick)
			}
		case "l":
			m.state = stateDownloads
			m.ActiveTab = 2
		case "c":
			m.state = stateConfig
			m.InitConfigInputs() // Refresh inputs
			m.ConfigFocusIndex = 0
		case "j":
			m.state = stateBatch
			m.FilePicker.CurrentDirectory, _ = os.Getwd() // Reset to cwd
			return m, m.FilePicker.Init()
		case "i":
			m.input.Focus()
			return m, textinput.Blink
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		
		// Resize lists
		m.Dialogs.SetSize(m.width/3, m.height-4)
		m.Messages.SetSize((m.width/3)*2, m.height-4)
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}
