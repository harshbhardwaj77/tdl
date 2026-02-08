package tui

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/spf13/viper"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/gotd/td/telegram"
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
	
	// Persistent Client
	Client     *telegram.Client
	ClientCtx  context.Context
	ClientCancel context.CancelFunc
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
	// Initialize Status
	m.StatusMessage = "Connecting to Telegram..."
	
	return tea.Batch(
		m.spinner.Tick,
		m.startClient, // Start the persistent connection
		m.GetAccounts(),
	)
}

func (m *Model) startClient() tea.Msg {
	// Cleanup existing client if any
	if m.ClientCancel != nil {
		m.ClientCancel()
	}

	// Create context for the client lifecycle
	m.ClientCtx, m.ClientCancel = context.WithCancel(context.Background())
	
	// Create the client instance 
	// We need tclient options
	opts := tclient.Options{
		KV: m.storage,
		// Add other options from config/viper if needed
	}
	
	var err error
	m.Client, err = tclient.New(m.ClientCtx, opts, false)
	if err != nil {
		return loginMsg{Err: err}
	}
	
	// Run the client in a goroutine
	// We use a channel to signal when the client is ready/authorized effectively?
	// Actually client.Run blocks. We need to run it and then perform a self check.
	// But checkLogin expects a return based on 'User'. 
	
	// Strategy:
	// 1. Start client.Run in goroutine.
	// 2. Wait for it to be ready (Auth Status).
	// 3. Fetch Self.
	// 4. Return loginMsg.
	
	readyCh := make(chan struct{})
	errCh := make(chan error)
	
	go func() {
		err := m.Client.Run(m.ClientCtx, func(ctx context.Context) error {
			// Signal ready
			close(readyCh)
			// Choose to block until context is done
			<-ctx.Done()
			return ctx.Err()
		})
		if err != nil && err != context.Canceled {
			errCh <- err
		}
	}()
	
	// Wait for ready or error
	select {
	case <-readyCh:
		// Client is running. Now check auth.
		// We need to use m.ClientCtx or a sub-context
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		status, err := m.Client.Auth().Status(ctx)
		if err != nil {
			return loginMsg{Err: err}
		}
		
		if !status.Authorized {
			return loginMsg{Err: fmt.Errorf("not authorized")}
		}
		
		// Fetch Self
		self, err := m.Client.Self(ctx)
		if err != nil {
			return loginMsg{Err: err}
		}
		return loginMsg{User: self}
		
	case err := <-errCh:
		return loginMsg{Err: err}
	case <-time.After(15 * time.Second):
		return loginMsg{Err: fmt.Errorf("connection timeout")}
	}
}
