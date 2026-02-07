package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/gotd/td/tg"
	
	"github.com/iyear/tdl/core/storage"
	"github.com/iyear/tdl/pkg/consts"
	"github.com/iyear/tdl/pkg/tclient"
)

type sessionState int

const (
	stateDashboard sessionState = iota
	stateDownloads
	stateLogin
)

type Model struct {
	state      sessionState
	width      int
	height     int
	quitting   bool
	
	// Components
	spinner    spinner.Model
	list       list.Model
	viewport   viewport.Model
	input      textinput.Model
	
	// Data
	Namespace  string
	Connected  bool
	BuildInfo  string
	User       *tg.User
	Downloads  map[string]*DownloadItem
	
	// Internal
	storage    storage.Storage
	tuiProgram *tea.Program
}

type loginMsg struct {
	User *tg.User
	Err  error
}

func NewModel(s storage.Storage) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	
	ti := textinput.New()
	ti.Placeholder = "Enter Telegram Link..."
	ti.CharLimit = 156
	ti.Width = 40

	return &Model{
		state:     stateDashboard,
		spinner:   sp,
		Namespace: consts.DefaultNamespace,
		BuildInfo: consts.Version,
		storage:   s,
		Downloads: make(map[string]*DownloadItem),
		input:     ti,
	}
}

func (m *Model) SetProgram(p *tea.Program) {
	m.tuiProgram = p
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.checkLogin,
	)
}

func (m *Model) checkLogin() tea.Msg {
	// Create a background context for the client
	// In a real app we might want to manage this context better
	ctx := context.Background()
	
	// We need to construct minimal options for tclient.New
	// We don't have full access to viper flags here easily unless we pass them or use viper directly
	// For now let's assume standard options. 
	// To do this properly we should pass Options to NewModel.
	// But let's try a simpler approach check: check if session exists in storage.
	
	// Actually, we can just try to create a client with existing session
	// We need 'tclient.Options' which requires KV.
	
	opts := tclient.Options{
		KV: m.storage,
		// We omit Proxy/NTP for this simple check or load from viper if needed
	}
	
	client, err := tclient.New(ctx, opts, false) // false = no interactive login
	if err != nil {
		return loginMsg{Err: err}
	}
	
	var user *tg.User
	err = client.Run(ctx, func(ctx context.Context) error {
		self, err := client.Self(ctx)
		if err != nil {
			return err
		}
		user = self
		return nil
	})
	
	if err != nil {
		return loginMsg{Err: err}
	}
	
	return loginMsg{User: user}
}
