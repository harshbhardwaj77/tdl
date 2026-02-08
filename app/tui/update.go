package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
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
			m.BatchPath = path
			m.state = stateBatchConfirm
			return m, nil
		}
		
		// Handle Esc to exit
		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "esc" {
			m.state = stateDashboard
		}
		
		return m, cmd
	}

	// Batch Confirm Intercept
	if m.state == stateBatchConfirm {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "d", "enter":
				m.state = stateDownloads
				m.ActiveTab = 2
				return m, m.startBatchDownload(m.BatchPath)
			case "f":
				m.PickingDest = true
				m.ForwardSource = []string{m.BatchPath}
				m.state = stateDashboard
				m.ActiveTab = 1 // Browser
				m.Pane = 0      // Dialogs
				m.StatusMessage = "Select destination chat for JSON batch..."
				// Trigger dialog fetch if needed
				if len(m.Dialogs.Items()) == 0 {
					m.LoadingDialogs = true
					return m, tea.Batch(m.GetDialogs(), m.spinner.Tick)
				}
				return m, nil
			case "esc", "q":
				m.state = stateBatch
				m.BatchPath = ""
				return m, nil
			}
		}
		return m, nil
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
				Path:     msg.Name, // Assuming msg.Name is the file path
				Total:    msg.Total,
				Progress: prog,
			}
			// Use Base name for display if it looks like a path
			if filepath.IsAbs(msg.Name) || len(filepath.Dir(msg.Name)) > 1 {
				// We keep Name as full path for map key? 
				// Actually typically msg.Name from TUIProgress is what we get.
				// Let's rely on Title() doing Base() if we want, or here.
				// For now simple.
			}
			
			m.Downloads[msg.Name] = item
			m.DownloadList.InsertItem(0, item)
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
	case AccountsMsg:
		if msg.Err == nil {
			m.Accounts = msg.Accounts
		}
		
	case AccountSwitchedMsg:
		if msg.Err != nil {
			m.StatusMessage = fmt.Sprintf("Switch Failed: %v", msg.Err)
		} else {
			m.Namespace = msg.Namespace
			m.storage = msg.Storage
			m.StatusMessage = fmt.Sprintf("Switched to %s", msg.Namespace)
			
			// Reset State
			m.User = nil
			m.Connected = false
			m.Dialogs.SetItems(nil)
			m.Messages.SetItems(nil)
			
			// Re-login
			return m, m.startClient
		}

	case tea.KeyMsg:
		// 1. Priority Globals
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "?":
			m.ShowHelp = !m.ShowHelp
			return m, nil
		case "esc":
			if m.ShowHelp {
				m.ShowHelp = false
				return m, nil
			}
			// History Pop
			if len(m.TabHistory) > 0 {
				last := m.TabHistory[len(m.TabHistory)-1]
				m.TabHistory = m.TabHistory[:len(m.TabHistory)-1]
				m.ActiveTab = last
				// Restore State
				switch last {
				case 0: m.state = stateDashboard
				case 1: m.state = stateDashboard // Browser shares state usually or we can verify
				case 2: m.state = stateDownloads
				}
				return m, nil
			}
			// Fallthrough to global quit checking if history empty?
			// Or just do nothing.
			
		}

		// 2. Global Navigation (Safe Keys)
		// j is excluded here to allow list navigation in Browser
		switch msg.String() {
		case "d":
			if m.ActiveTab != 0 {
				m.TabHistory = append(m.TabHistory, m.ActiveTab)
				m.state = stateDashboard
				m.ActiveTab = 0
			}
			return m, nil
		case "b":
			if m.ActiveTab != 1 {
				m.TabHistory = append(m.TabHistory, m.ActiveTab)
				m.ActiveTab = 1
				if len(m.Dialogs.Items()) == 0 {
					m.LoadingDialogs = true
					return m, tea.Batch(m.GetDialogs(), m.spinner.Tick)
				}
			}
			return m, nil
		case "l":
			if m.ActiveTab != 2 {
				m.TabHistory = append(m.TabHistory, m.ActiveTab)
				m.state = stateDownloads
				m.ActiveTab = 2
			}
			return m, nil
		case "c":
			m.state = stateConfig
			m.InitConfigInputs()
			m.ConfigFocusIndex = 0
			return m, nil
		case "i":
			m.input.Focus()
			return m, textinput.Blink
		case "a":
			if len(m.Accounts) > 1 {
				idx := -1
				for i, acc := range m.Accounts {
					if acc == m.Namespace { idx = i; break }
				}
				nextIdx := (idx + 1) % len(m.Accounts)
				return m, m.SwitchAccount(m.Accounts[nextIdx])
			}
			return m, nil
		case "tab":
			if m.ActiveTab == 1 {
				m.Pane = 1 - m.Pane
				return m, nil
			}
		}

		// If input is focused, pass messages to it
		if m.input.Focused() {
			switch msg.String() {
			case "enter":
				val := m.input.Value()
				m.input.Reset()
				m.input.Blur()
				
				if m.Searching {
					m.Searching = false
					m.LoadingDialogs = true
					m.Pane = 0
					m.StatusMessage = "Searching: " + val
					return m, tea.Batch(m.SearchPeers(val), m.spinner.Tick)
				}

				m.ActiveTab = 2 // Switch to downloads
				return m, m.startDownload(val)
			case "esc":
				m.Searching = false
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
				// Handle Enter
				if msg.String() == "enter" {
					if m.PickingDest {
						// Execute Forward
						if dlg, ok := m.Dialogs.SelectedItem().(DialogItem); ok {
							dest := strconv.FormatInt(dlg.PeerID, 10) // Use ID as dest
							sources := m.ForwardSource
							
							// Reset state
							m.PickingDest = false
							m.ForwardSource = nil
							m.StatusMessage = fmt.Sprintf("Forwarding to %s...", dlg.Title)
							
							return m, m.startForward(dest, sources)
						}
					}

					// Fetch history for selected dialog
					if dlg, ok := m.Dialogs.SelectedItem().(DialogItem); ok {
						m.Messages.SetItems(nil) // Clear previous
						m.LoadingHistory = true
						return m, tea.Batch(m.GetHistory(dlg.Peer), m.spinner.Tick)
					}
					m.Pane = 1
				}
				// Handle Export
				if msg.String() == "e" && !m.PickingDest {
					if dlg, ok := m.Dialogs.SelectedItem().(DialogItem); ok {
						m.LoadingExport = true
						return m, tea.Batch(m.startExport(dlg), m.spinner.Tick)
					}
				}
				return m, cmd
			} else {
				m.Messages, cmd = m.Messages.Update(msg)
				
				// Message Selection (Space)
				if msg.String() == " " {
					if idx := m.Messages.Index(); idx >= 0 {
						if item, ok := m.Messages.SelectedItem().(MessageItem); ok {
							item.Selected = !item.Selected
							m.Messages.SetItem(idx, item)
							return m, nil
						}
					}
				}

				// Forward Init (f)
				if msg.String() == "f" {
					// ... (existing forward logic)
					// Collect selected
					var sources []string
					for _, item := range m.Messages.Items() {
						if mItem, ok := item.(MessageItem); ok && mItem.Selected {
							// Construct link
							link := fmt.Sprintf("https://t.me/c/%d/%d", mItem.ChatID, mItem.ID)
							sources = append(sources, link)
						}
					}
					
					if len(sources) > 0 {
						m.PickingDest = true
						m.ForwardSource = sources
						m.Pane = 0 // Switch to dialogs to pick
						m.StatusMessage = fmt.Sprintf("Select destination chat for %d messages...", len(sources))
						return m, nil
					}
					m.StatusMessage = "No messages selected. Use [Space] to select."
					return m, nil
				}

				if msg.String() == "enter" {
					// ... (download logic)
				}
				return m, cmd
			}
		}

		switch msg.String() {
		case "j":
			m.state = stateBatch
			m.FilePicker.CurrentDirectory, _ = os.Getwd() // Reset to cwd
			return m, m.FilePicker.Init()
		case "s":
			if m.ActiveTab == 1 { // Browser
				m.Searching = true
				m.input.Placeholder = "Search Global... (Enter to submit)"
				m.input.Focus()
				return m, textinput.Blink
			}
		}
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// Tab Hit Testing (Approximate)
			// Header is ~2 lines, padding ~1 line. Tabs usually around line 3-5.
			if msg.Y >= 2 && msg.Y <= 6 {
				// Tabs are left aligned: Dashboard | Browser | Downloads
				// Dashboard ~12 chars, Browser ~10 chars, Downloads ~12 chars
				// Padding adds to it.
				if msg.X >= 0 && msg.X < 15 {
					if m.ActiveTab != 0 {
						m.TabHistory = append(m.TabHistory, m.ActiveTab)
						m.ActiveTab = 0
						m.state = stateDashboard
					}
					return m, nil
				} else if msg.X >= 15 && msg.X < 30 {
					if m.ActiveTab != 1 {
						m.TabHistory = append(m.TabHistory, m.ActiveTab)
						m.ActiveTab = 1
						// Trigger fetch dialogs if empty
						if len(m.Dialogs.Items()) == 0 {
							m.LoadingDialogs = true
							return m, tea.Batch(m.GetDialogs(), m.spinner.Tick)
						}
					}
					return m, nil
				} else if msg.X >= 30 && msg.X < 50 {
					if m.ActiveTab != 2 {
						m.TabHistory = append(m.TabHistory, m.ActiveTab)
						m.ActiveTab = 2
						m.state = stateDownloads
					}
					return m, nil
				}
			}
		}
		
		// Forward mouse to active components
		var cmd tea.Cmd
		if m.ActiveTab == 1 {
			if m.Pane == 0 {
				m.Dialogs, cmd = m.Dialogs.Update(msg)
				return m, cmd
			} else {
				m.Messages, cmd = m.Messages.Update(msg)
				return m, cmd
			}
		} else if m.ActiveTab == 2 {
			var cmd tea.Cmd
			m.DownloadList, cmd = m.DownloadList.Update(msg)
			
			if msg.String() == "o" {
				if item, ok := m.DownloadList.SelectedItem().(*DownloadItem); ok {
					if err := openFile(item.Path); err != nil {
						m.StatusMessage = fmt.Sprintf("Failed to open: %v", err)
					} else {
						m.StatusMessage = fmt.Sprintf("Opening %s...", filepath.Base(item.Path))
					}
				}
			}
			return m, cmd
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		
		// Resize lists
		m.Dialogs.SetSize(m.width/3, m.height-4)
		m.Messages.SetSize((m.width/3)*2, m.height-4)
		m.DownloadList.SetSize(m.width, m.height-4)

	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}
