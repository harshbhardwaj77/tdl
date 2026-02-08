package tui

import (
	"context"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/spf13/viper"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/gotd/td/tg"
	"github.com/iyear/tdl/pkg/tclient"
	"github.com/iyear/tdl/pkg/kv"
	
	"github.com/iyear/tdl/core/storage"
	"github.com/iyear/tdl/pkg/consts"
)

type sessionState int

const (
	stateDashboard sessionState = iota
	stateDownloads
	stateConfig
	stateBrowser
	stateBatch
	stateBatchConfirm
	stateLogin
)

type Model struct {
	state      sessionState
	ActiveTab  int // 0: Dashboard, 1: Browser, 2: Downloads
	TabHistory []int // Navigation stack for Esc key
	
	// Browser State
	Dialogs    list.Model
	Messages   list.Model
	Browsing   bool // True if focused heavily on browser
	Pane       int // 0: Dialogs (Left), 1: Messages (Right)
	SelectedApp *tclient.App
	LoadingDialogs bool
	LoadingHistory bool
	LoadingExport  bool
	Searching      bool // Global Search input mode
	
	// Forwarding
	PickingDest    bool // If true, selecting a dialog = forward destination
	ForwardSource  []string
	
	// UI State
	ShowHelp       bool


	
	width      int
	height     int
	quitting   bool
	
	// Components
	spinner    spinner.Model
	list       list.Model
	viewport   viewport.Model
	input      textinput.Model
	
	// Config Editor
	ConfigInputs     []textinput.Model
	ConfigFocusIndex int
	
	// Batch Processing
	FilePicker filepicker.Model
	BatchPath  string
	
	// Data
	Namespace  string
	Connected  bool
	BuildInfo  string
	User       *tg.User
	Downloads  map[string]*DownloadItem
	DownloadList list.Model // New list for downloads
	StatusMessage string
	
	// Account Management
	Accounts   []string
	kvStorage  kv.Storage
	
	// Internal
	storage    storage.Storage
	tuiProgram *tea.Program
}

type loginMsg struct {
	User *tg.User
	Err  error
}

type ExportMsg struct {
	Path string
	Err  error
}

type AccountsMsg struct {
	Accounts []string
	Err      error
}

type AccountSwitchedMsg struct {
	Namespace string
	Storage   storage.Storage
	Err       error
}

func NewModel(root kv.Storage, s storage.Storage, ns string) *Model {
	// Initialize Theme
	p := viper.GetString("theme.primary")
	sec := viper.GetString("theme.secondary")
	errColor := viper.GetString("theme.error")
	suc := viper.GetString("theme.success")
	dim := viper.GetString("theme.dim")

	// Fallback defaults if empty
	if p == "" { p = "62" }
	if sec == "" { sec = "230" }
	if errColor == "" { errColor = "196" }
	if suc == "" { suc = "42" }
	if dim == "" { dim = "240" }

	InitStyles(p, sec, errColor, suc, dim)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	
	ti := textinput.New()
	ti.Placeholder = "Enter Telegram Link..."
	ti.CharLimit = 156
	ti.Width = 40

	// Initialize Browser Lists
	dList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	dList.Title = "Chats"
	dList.SetShowHelp(false)
	
	mList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	mList.Title = "Messages"
	mList.SetShowHelp(false)

	// File Picker
	fp := filepicker.New()
	fp.AllowedTypes = []string{".json"}
	fp.CurrentDirectory, _ = os.Getwd()

	// Download List
	dlList := list.New([]list.Item{}, ItemDelegate{}, 0, 0)
	dlList.Title = "Downloads"
	dlList.SetShowHelp(false)

	return &Model{
		state:     stateDashboard,
		ActiveTab: 0, // Dashboard default
		Dialogs:   dList,
		Messages:  mList,
		Pane:      0, // Start with Dialogs focused
		spinner:   sp,
		Namespace: ns,
		BuildInfo: consts.Version,
		storage:   s,
		kvStorage: root,
		Downloads: make(map[string]*DownloadItem),
		DownloadList: dlList,
		input:     ti,
		FilePicker: fp,
	}
}

func (m *Model) SetProgram(p *tea.Program) {
	m.tuiProgram = p
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.checkLogin,
		m.GetAccounts(),
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
