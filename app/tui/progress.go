package tui

import (
	"fmt"
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
	
	if err == nil {
		// Send notification
		// We run this in a goroutine to avoid blocking
		go func() {
			notify("Download Complete", fmt.Sprintf("%s has finished downloading.", name))
		}()
	}
}

// DownloadItem represents a single download in the list
type DownloadItem struct {
	Name       string
	Path       string // Full absolute path
	Total      int64
	Downloaded int64
	Progress   progress.Model
	Finished   bool
	Err        error
}

func (d *DownloadItem) Title() string {
	return d.Name
}

func (d *DownloadItem) Description() string {
	if d.Finished {
		if d.Err != nil {
			return "❌ Failed: " + d.Err.Error()
		}
		return "✅ Completed"
	}
	
	prog := d.Progress.ViewAs(d.percent())
	return fmt.Sprintf("%s", prog)
}

func (d *DownloadItem) FilterValue() string {
	return d.Name
}

func (d *DownloadItem) percent() float64 {
	if d.Total <= 0 {
		return 0
	}
	return float64(d.Downloaded) / float64(d.Total)
}
