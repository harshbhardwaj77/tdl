package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/viper"
	"github.com/iyear/tdl/pkg/consts"
)

// Config Keys (Must match flags)
var configKeys = []string{
	consts.FlagNamespace,
	consts.FlagProxy,
	consts.FlagThreads,
	consts.FlagLimit,
	consts.FlagPartSize,
	consts.FlagDelay,
	consts.FlagReconnectTimeout,
	consts.FlagDlTemplate,
	"download_dir", // Custom key for TUI
	"group",
	"skip_same",
	"takeout",
	"continue",
	"theme.primary",
	"theme.secondary",
	"theme.error",
	"theme.success",
	"theme.dim",
	"notify",
}

func (m *Model) InitConfigInputs() {
	m.ConfigInputs = make([]textinput.Model, len(configKeys))
	for i := range m.ConfigInputs {
		t := textinput.New()
		t.Cursor.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
		t.Prompt = configKeys[i] + ": "
		t.PromptStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
		
		// Load initial value
		val := viper.GetString(configKeys[i])
		if configKeys[i] == "download_dir" && val == "" {
			val = "downloads"
		}
		t.SetValue(val)
		
		m.ConfigInputs[i] = t
	}
}

func (m *Model) SaveConfig() error {
	for i, input := range m.ConfigInputs {
		key := configKeys[i]
		val := input.Value()
		viper.Set(key, val)
	}
	
	// Apply Theme immediately
	p := viper.GetString("theme.primary")
	sec := viper.GetString("theme.secondary")
	errColor := viper.GetString("theme.error")
	suc := viper.GetString("theme.success")
	dim := viper.GetString("theme.dim")
	if p == "" { p = "62" }
	if sec == "" { sec = "230" }
	if errColor == "" { errColor = "196" }
	if suc == "" { suc = "42" }
	if dim == "" { dim = "240" }
	InitStyles(p, sec, errColor, suc, dim)

	return viper.WriteConfigAs("tdl.toml")
}

func (m *Model) updateConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "up", "down":
			s := msg.String()
			
			if s == "up" || s == "shift+tab" {
				m.ConfigFocusIndex--
			} else {
				m.ConfigFocusIndex++
			}

			if m.ConfigFocusIndex > len(m.ConfigInputs) { // +1 for Save button
				m.ConfigFocusIndex = 0
			} else if m.ConfigFocusIndex < 0 {
				m.ConfigFocusIndex = len(m.ConfigInputs)
			}
			
			cmds := make([]tea.Cmd, len(m.ConfigInputs))
			for i := 0; i < len(m.ConfigInputs); i++ {
				if i == m.ConfigFocusIndex {
					// Set focused state
					cmds[i] = m.ConfigInputs[i].Focus()
					m.ConfigInputs[i].PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimary)
				} else {
					m.ConfigInputs[i].Blur()
					m.ConfigInputs[i].PromptStyle = lipgloss.NewStyle().Foreground(ColorSecondary)
				}
			}
			return m, tea.Batch(cmds...)
			
		case "enter":
			if m.ConfigFocusIndex == len(m.ConfigInputs) {
				// Save button clicked
				if err := m.SaveConfig(); err != nil {
					// handle error
				}
				// Go back to dashboard
				m.state = stateDashboard
				return m, nil
			}
			
		case "esc":
			m.state = stateDashboard
			return m, nil
		}
	}

	// Update inputs
	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *Model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.ConfigInputs))
	for i := range m.ConfigInputs {
		m.ConfigInputs[i], cmds[i] = m.ConfigInputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m *Model) viewConfig() string {
	var s strings.Builder
	s.WriteString(TitleStyle.Render("Configuration Editor") + "\n\n")

	for i := range m.ConfigInputs {
		s.WriteString(m.ConfigInputs[i].View())
		s.WriteString("\n")
	}

	s.WriteString("\n")
	
	// Save Button
	btn := "[ Save Changes ]"
	if m.ConfigFocusIndex == len(m.ConfigInputs) {
		btn = ActivePaneStyle.Render(btn)
	} else {
		btn = InactivePaneStyle.Render(btn)
	}
	s.WriteString(btn)
	s.WriteString("\n\n")
	s.WriteString(StatusBarStyle.Render("  [Tab] Next Field • [Enter] Save/Next • [Esc] Cancel"))

	return s.String()
}
