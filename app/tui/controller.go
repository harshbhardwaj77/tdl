package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iyear/tdl/app/dl"
	"github.com/iyear/tdl/core/logctx"
	"github.com/iyear/tdl/pkg/tclient"
)

func (m *Model) startDownload(url string) tea.Cmd {
	return func() tea.Msg {
		if url == "" {
			return nil
		}
		
		// In a real app we'd want to manage context cancellation
		ctx := context.Background() 
		
		// Prepare Options
		opts := dl.Options{
			URLs: []string{url},
			Dir:  "downloads", // Default to downloads dir
			// Set other defaults as needed
		}
		
		// We need to run this in a way that respects the existing architecture
		// The key challenge is that dl.Run takes existing Client and KV
		// We have KV, but Client is usually created inside tRun or passed in.
		// In our login check we created a client briefly.
		// We should probably keep a persistent client or recreate it.
		// Recreating it is safer for now.
		
		tOpts := tclient.Options{
			KV: m.storage,
		}
		
		client, err := tclient.New(ctx, tOpts, false)
		if err != nil {
			return ProgressMsg{Name: url, Err: err, IsFinished: true}
		}
		
		return client.Run(ctx, func(ctx context.Context) error {
			// Inject TUI progress and enable Silent mode
			opts.Silent = true
			opts.ExternalProgress = NewTUIProgress(m.tuiProgram) // We need program ref
			
			// We need to access kvd from tclient options or pass it
			// dl.Run needs storage.Storage.
			// m.storage is available.
			
			return dl.Run(logctx.Named(ctx, "dl"), client, m.storage, opts)
		})
	}
}
