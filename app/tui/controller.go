package tui

import (
	"context"
	"strconv"
	"math"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/spf13/viper"
	"github.com/iyear/tdl/app/dl"
	"github.com/iyear/tdl/app/chat"
	"github.com/iyear/tdl/core/logctx"
	"github.com/iyear/tdl/pkg/consts"
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
		dir := viper.GetString("download_dir")
		if dir == "" {
			dir = "downloads"
		}
		
		opts := dl.Options{
			URLs: []string{url},
			Dir:  dir,
			Template: viper.GetString(consts.FlagDlTemplate),
			Group:    viper.GetBool("group"),
			SkipSame: viper.GetBool("skip_same"),
			Takeout:  viper.GetBool("takeout"),
			Continue: viper.GetBool("continue"),
		}
		
		// We need to run this in a way that respects the existing architecture
		// The key challenge is that dl.Run takes existing Client and KV
		// We have KV, but Client is usually created inside tRun or passed in.
		// In our login check we created a client briefly.
		// We should probably keep a persistent client or recreate it.
		// Recreating it is safer for now.
		
		tOpts := tclient.Options{
			KV:               m.storage,
			Proxy:            viper.GetString(consts.FlagProxy),
			NTP:              viper.GetString(consts.FlagNTP),
			ReconnectTimeout: viper.GetDuration(consts.FlagReconnectTimeout),
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

func (m *Model) startBatchDownload(path string) tea.Cmd {
	return func() tea.Msg {
		if path == "" {
			return nil
		}
		
		ctx := context.Background() 
		
		// Prepare Options (Respected Config)
		dir := viper.GetString("download_dir")
		if dir == "" {
			dir = "downloads"
		}
		
		opts := dl.Options{
			Files: []string{path}, // Use Files instead of URLs
			Dir:   dir,
			Template: viper.GetString(consts.FlagDlTemplate),
			Group:    viper.GetBool("group"),
			SkipSame: viper.GetBool("skip_same"),
			Takeout:  viper.GetBool("takeout"),
			Continue: viper.GetBool("continue"),
		}
		
		tOpts := tclient.Options{
			KV:               m.storage,
			Proxy:            viper.GetString(consts.FlagProxy),
			NTP:              viper.GetString(consts.FlagNTP),
			ReconnectTimeout: viper.GetDuration(consts.FlagReconnectTimeout),
		}
		
		client, err := tclient.New(ctx, tOpts, false)
		if err != nil {
			return ProgressMsg{Name: path, Err: err, IsFinished: true}
		}
		
		return client.Run(ctx, func(ctx context.Context) error {
			opts.Silent = true
			opts.ExternalProgress = NewTUIProgress(m.tuiProgram)
			
			return dl.Run(logctx.Named(ctx, "dl"), client, m.storage, opts)
		})
	}
}

func (m *Model) startExport(d DialogItem) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Use chat ID as filename: {ID}.json
		filename := fmt.Sprintf("%d.json", d.PeerID)
		
		// Setup Options
		opts := chat.ExportOptions{
			Type:   chat.ExportTypeTime,
			Input:  []int{0, math.MaxInt}, // All history
			Output: filename,
			Chat:   strconv.FormatInt(d.PeerID, 10),
			Silent: true,
			Filter: "true",
		}

		tOpts := tclient.Options{
			KV:               m.storage,
			Proxy:            viper.GetString(consts.FlagProxy),
			NTP:              viper.GetString(consts.FlagNTP),
			ReconnectTimeout: viper.GetDuration(consts.FlagReconnectTimeout),
		}

		client, err := tclient.New(ctx, tOpts, false)
		if err != nil {
			return ExportMsg{Err: err}
		}

		err = client.Run(ctx, func(ctx context.Context) error {
			return chat.Export(logctx.Named(ctx, "export"), client, m.storage, opts)
		})
		
		return ExportMsg{Path: filename, Err: err}
	}
}
