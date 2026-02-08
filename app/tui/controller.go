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
	"github.com/iyear/tdl/app/forward"
	"github.com/gotd/td/tg"
	"github.com/iyear/tdl/core/forwarder"
	"github.com/iyear/tdl/core/logctx"
	"github.com/iyear/tdl/pkg/consts"
	"github.com/iyear/tdl/pkg/tclient"
)

func (m *Model) startDownload(url string) tea.Cmd {
	storage := m.storage // Capture for thread safety
	prog := m.tuiProgram
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
			opts.ExternalProgress = NewTUIProgress(prog)
			
			return dl.Run(logctx.Named(ctx, "dl"), client, storage, opts)
		})
	}
}

func (m *Model) startBatchDownload(path string) tea.Cmd {
	storage := m.storage
	prog := m.tuiProgram
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
			KV:               storage,
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
			opts.ExternalProgress = NewTUIProgress(prog)
			return dl.Run(logctx.Named(ctx, "dl"), client, storage, opts)
		})
	}
}

func (m *Model) startExport(d DialogItem) tea.Cmd {
	storage := m.storage
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
			KV:               storage,
			Proxy:            viper.GetString(consts.FlagProxy),
			NTP:              viper.GetString(consts.FlagNTP),
			ReconnectTimeout: viper.GetDuration(consts.FlagReconnectTimeout),
		}

		client, err := tclient.New(ctx, tOpts, false)
		if err != nil {
			return ExportMsg{Err: err}
		}

		err = client.Run(ctx, func(ctx context.Context) error {
			return chat.Export(logctx.Named(ctx, "export"), client, storage, opts)
		})
		
		return ExportMsg{Path: filename, Err: err}
	}
}

func (m *Model) GetAccounts() tea.Cmd {
	return func() tea.Msg {
		if m.kvStorage == nil {
			return nil
		}
		items, err := m.kvStorage.Namespaces()
		if err != nil {
			return AccountsMsg{Err: err}
		}
		return AccountsMsg{Accounts: items}
	}
}

func (m *Model) SwitchAccount(ns string) tea.Cmd {
	return func() tea.Msg {
		if m.kvStorage == nil {
			return nil
		}
		kvd, err := m.kvStorage.Open(ns)
		return AccountSwitchedMsg{Namespace: ns, Storage: kvd, Err: err}
	}
}

func (m *Model) startForward(dest string, sources []string) tea.Cmd {
	storage := m.storage
	return func() tea.Msg {
		ctx := context.Background()
		
		opts := forward.Options{
			From:   sources,
			To:     dest, // Destination is now dynamic
			Mode:   forwarder.ModeClone,
			Silent: true,
		}

		tOpts := tclient.Options{
			KV:               storage,
			Proxy:            viper.GetString(consts.FlagProxy),
			NTP:              viper.GetString(consts.FlagNTP),
			ReconnectTimeout: viper.GetDuration(consts.FlagReconnectTimeout),
		}

		client, err := tclient.New(ctx, tOpts, false)
		if err != nil {
			return ExportMsg{Err: fmt.Errorf("client init: %w", err)} // Reuse msg or new one?
		}

		err = client.Run(ctx, func(ctx context.Context) error {
			return forward.Run(logctx.Named(ctx, "forward"), client, storage, opts)
		})
		return ExportMsg{Path: "Forwarded", Err: err} // Reusing ExportMsg for simplicity for now
	}
}

func (m *Model) SearchPeers(query string) tea.Cmd {
	storage := m.storage
	return func() tea.Msg {
		if query == "" { return nil }
		ctx := context.Background()

		tOpts := tclient.Options{
			KV:               storage,
			Proxy:            viper.GetString(consts.FlagProxy),
			NTP:              viper.GetString(consts.FlagNTP),
			ReconnectTimeout: viper.GetDuration(consts.FlagReconnectTimeout),
		}

		client, err := tclient.New(ctx, tOpts, false)
		if err != nil {
			return dialogsMsg{Err: err}
		}

		var results []DialogItem

		err = client.Run(ctx, func(ctx context.Context) error {
			res, err := client.API().ContactsSearch(ctx, &tg.ContactsSearchRequest{
				Q:     query,
				Limit: 20,
			})
			if err != nil {
				return err
			}

			// gotd might return concrete type for ContactsSearch
			found := res
			// Check if it's interface or struct
			// If it was interface, previous code would work.
			// Error said "is not an interface", so it's struct.
			// We assume found = res works and has .Users etc.
			
			// Note: if found is *ContactsFound, it works.
			
			// Helper to find title and input peer
			getTitle := func(peerC tg.PeerClass) (string, int64, tg.InputPeerClass) {
				var id int64
				var title string
				var inputPeer tg.InputPeerClass

				switch p := peerC.(type) {
				case *tg.PeerUser:
					id = p.UserID
					for _, u := range found.Users {
						switch user := u.(type) {
						case *tg.User:
							if user.ID == id {
								title = user.FirstName + " " + user.LastName
								if user.Username != "" { title += " (@" + user.Username + ")" }
								inputPeer = &tg.InputPeerUser{UserID: id, AccessHash: user.AccessHash}
							}
						}
						if inputPeer != nil { break }
					}
					if inputPeer == nil { inputPeer = &tg.InputPeerUser{UserID: id} } // Fallback
					
				case *tg.PeerChat:
					id = p.ChatID
					for _, c := range found.Chats {
						switch chat := c.(type) {
						case *tg.Chat:
							if chat.ID == id {
								title = chat.Title
							}
						}
						// Chat usually doesn't need access hash for InputPeerChat
						if title != "" { break }
					}
					inputPeer = &tg.InputPeerChat{ChatID: id}
					
				case *tg.PeerChannel:
					id = p.ChannelID
					for _, c := range found.Chats {
						switch chat := c.(type) {
						case *tg.Channel:
							if chat.ID == id {
								title = chat.Title
								inputPeer = &tg.InputPeerChannel{ChannelID: id, AccessHash: chat.AccessHash}
							}
						}
						if inputPeer != nil { break }
					}
					if inputPeer == nil { inputPeer = &tg.InputPeerChannel{ChannelID: id} }
				}
				
				if title == "" { title = fmt.Sprintf("Unknown#%d", id) }
				return title, id, inputPeer
			}

			// Process Results
			for _, p := range found.Results {
				title, id, inputPeer := getTitle(p)
				results = append(results, DialogItem{
					Title:  title,
					PeerID: id,
					Peer:   inputPeer,
				})
			}
			return nil
		})

		return dialogsMsg{Dialogs: results, Err: err}
	}
}
