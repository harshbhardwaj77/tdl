package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/iyear/tdl/core/downloader"
)

// ProgressMsg updates the TUI with download progress
type ProgressMsg struct {
	ID         int64 // Unique ID for the download (using message ID or similar)
	Name       string
	State      downloader.ProgressState
	Total      int64
	IsFinished bool
	Err        error
}

// Ensure Model satisfies downloader.Progress interface
// Note: We need a structural adapter because Model is a value receiver in View/Update usually, 
// and we need to send messages to the Program.
type TUIProgress struct {
	program *tea.Program
}

func NewTUIProgress(p *tea.Program) *TUIProgress {
	return &TUIProgress{program: p}
}

func (t *TUIProgress) OnAdd(elem downloader.Elem) {
	// Send initial add message
	// We need to extract ID/Name from elem
	// elem is likely *iterElem which has .fromMsg.ID
	// But Elem interface is:
	// File() *telegram.Document
	// To() *os.File
	// ...
	
	// We'll use the file name as key for now or just broadcast
	name := "unknown"
	if f, ok := elem.To().(interface{ Name() string }); ok {
		name = f.Name()
	}

	t.program.Send(ProgressMsg{
		Name:  name,
		Total: elem.File().Size(),
	})
}

func (t *TUIProgress) OnDownload(elem downloader.Elem, state downloader.ProgressState) {
	name := "unknown"
	if f, ok := elem.To().(interface{ Name() string }); ok {
		name = f.Name()
	}

	t.program.Send(ProgressMsg{
		Name:  name,
		State: state,
		Total: elem.File().Size(),
	})
}

func (t *TUIProgress) OnDone(elem downloader.Elem, err error) {
	name := "unknown"
	if f, ok := elem.To().(interface{ Name() string }); ok {
		name = f.Name()
	}

	t.program.Send(ProgressMsg{
		Name:       name,
		IsFinished: true,
		Err:        err,
	})
}

// DownloadItem represents a single download in the list
type DownloadItem struct {
	Name       string
	Total      int64
	Downloaded int64
	Progress   progress.Model
	Finished   bool
	Err        error
}
