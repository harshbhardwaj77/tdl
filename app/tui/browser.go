package tui

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tg"
	
	"github.com/iyear/tdl/pkg/tclient"
)

// Messages
type dialogsMsg struct {
	Dialogs []DialogItem
	Err     error
}

type historyMsg struct {
	Messages []MessageItem
	Err      error
}

// Items for List
type DialogItem struct {
	Title    string
	PeerID   int64
	Peer     tg.InputPeerClass
	Unread   int
	LastDate int // timestamp
}

func (d DialogItem) FilterValue() string { return d.Title }
func (d DialogItem) TitleString() string { return d.Title }
func (d DialogItem) Description() string { 
	return fmt.Sprintf("ID: %d | Unread: %d", d.PeerID, d.Unread) 
}

type MessageItem struct {
	ID       int
	ChatID   int64
	Peer     tg.InputPeerClass
	Text     string
	Date     int
	HasMedia bool
	Media    string
	File     *tg.InputFileLocation
	From     string
	Selected bool
}

func (m MessageItem) FilterValue() string { return m.Text }
func (m MessageItem) TitleString() string {
	prefix := " "
	if m.Selected {
		prefix = "[x] "
	}
	if m.HasMedia {
		return fmt.Sprintf("%s[%s] %s", prefix, m.Media, m.Text)
	}
	return prefix + m.Text 
}
func (m MessageItem) Description() string { 
	t := time.Unix(int64(m.Date), 0)
	return fmt.Sprintf("%s | ID: %d", t.Format("15:04 Jan 02"), m.ID)
}

// Commands
func logToFile(msg string) {
	f, _ := os.OpenFile("tui_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer f.Close()
	f.WriteString(time.Now().Format(time.RFC3339) + ": " + msg + "\n")
}

func (m *Model) GetDialogs() tea.Cmd {
	return func() tea.Msg {
		logToFile("GetDialogs: Starting")
		
		m.clientMu.Lock()
		client := m.Client
		m.clientMu.Unlock()
		
		if client == nil {
			logToFile("GetDialogs: Client is nil")
			return dialogsMsg{Err: fmt.Errorf("client not connected")}
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		
		// Use persistent client
		raw := tg.NewClient(client)
		
		logToFile("GetDialogs: Fetching API")
		dlgRes, err := raw.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			Limit: 20,
		})
		if err != nil {
			logToFile("GetDialogs: API Error: " + err.Error())
			return dialogsMsg{Err: err}
		}
		logToFile(fmt.Sprintf("GetDialogs: API Success, Type: %T", dlgRes))
		
		// Process response
		var (
			dialogs []tg.DialogClass
			chats   []tg.ChatClass
			users   []tg.UserClass
		)
		
		switch d := dlgRes.(type) {
		case *tg.MessagesDialogs:
			dialogs = d.Dialogs
			chats = d.Chats
			users = d.Users
		case *tg.MessagesDialogsSlice:
			dialogs = d.Dialogs
			chats = d.Chats
			users = d.Users
		}
		
		logToFile(fmt.Sprintf("GetDialogs: Found %d dialogs", len(dialogs)))

		// Map peers
		peerMap := make(map[int64]string)
		for _, u := range users {
			user := u.(*tg.User)
			peerMap[user.ID] = user.FirstName + " " + user.LastName
		}
		for _, c := range chats {
			switch chat := c.(type) {
			case *tg.Chat:
				peerMap[chat.ID] = chat.Title
			case *tg.Channel:
				peerMap[chat.ID] = chat.Title
			}
		}
		
		var items []DialogItem
		for _, d := range dialogs {
			dlg, ok := d.(*tg.Dialog)
			if !ok {
				continue
			}
			
			var peerID int64
			var title string
			var inputPeer tg.InputPeerClass
			
			switch p := dlg.Peer.(type) {
			case *tg.PeerUser:
				peerID = p.UserID
				title = peerMap[peerID]
				inputPeer = &tg.InputPeerUser{UserID: peerID}
				for _, u := range users {
					if user, ok := u.(*tg.User); ok && user.ID == peerID {
						inputPeer = &tg.InputPeerUser{UserID: peerID, AccessHash: user.AccessHash}
						break
					}
				}
			case *tg.PeerChat:
				peerID = p.ChatID
				title = peerMap[peerID]
				inputPeer = &tg.InputPeerChat{ChatID: peerID}
			case *tg.PeerChannel:
				peerID = p.ChannelID
				title = peerMap[peerID]
				for _, c := range chats {
					if ch, ok := c.(*tg.Channel); ok && ch.ID == peerID {
						inputPeer = &tg.InputPeerChannel{ChannelID: peerID, AccessHash: ch.AccessHash}
						if title == "" { title = ch.Title }
						break
					}
				}
				if inputPeer == nil { inputPeer = &tg.InputPeerChannel{ChannelID: peerID} }
			}
			
			if title == "" {
				title = fmt.Sprintf("Unknown Chat %d", peerID)
			}

			items = append(items, DialogItem{
				Title:    title,
				PeerID:   peerID,
				Unread:   dlg.UnreadCount,
				Peer:     inputPeer,
			})
		}
		
		logToFile(fmt.Sprintf("GetDialogs: Finished with %d items", len(items)))
		return dialogsMsg{Dialogs: items}
	}
}

func (m *Model) GetHistory(peer tg.InputPeerClass) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		opts := tclient.Options{KV: m.storage}
		client, err := tclient.New(ctx, opts, false)
		if err != nil {
			return historyMsg{Err: err}
		}

		var items []MessageItem
		
		err = client.Run(ctx, func(ctx context.Context) error {
			raw := tg.NewClient(client)
			
			// Get History
			histRes, err := raw.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
				Peer: peer,
				Limit: 50,
			})
			if err != nil {
				return err
			}
			
			// Resolve PeerID for link construction
			var peerID int64
			switch p := peer.(type) {
			case *tg.InputPeerChannel:
				peerID = p.ChannelID
			case *tg.InputPeerChat:
				peerID = p.ChatID
			case *tg.InputPeerUser:
				peerID = p.UserID
			}
			
			var messages []tg.MessageClass
			switch h := histRes.(type) {
			case *tg.MessagesMessages:
				messages = h.Messages
			case *tg.MessagesMessagesSlice:
				messages = h.Messages
			case *tg.MessagesChannelMessages:
				messages = h.Messages
			}
			
			for _, msg := range messages {
				switch m := msg.(type) {
				case *tg.Message:
					text := m.Message
					hasMedia := false
					mediaType := ""
					
					// Basic media check
					if m.Media != nil {
						hasMedia = true
						switch m.Media.(type) {
						case *tg.MessageMediaPhoto:
							mediaType = "Photo"
						case *tg.MessageMediaDocument:
							mediaType = "Document"
						default:
							mediaType = "Media"
						}
					}
					
					items = append(items, MessageItem{
						ID:       m.ID,
						ChatID:   peerID,
						Peer:     peer,
						Text:     text,
						Date:     m.Date,
						HasMedia: hasMedia,
						Media:    mediaType,
					})
				}
			}
			return nil
		})
		
		if err != nil {
			return historyMsg{Err: err}
		}
		
		return historyMsg{Messages: items}
	}
}
