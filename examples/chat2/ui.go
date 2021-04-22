package main

import (
	"chat2/pb"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/status-im/go-waku/waku/v2/node"
)

// ChatUI is a Text User Interface (TUI) for a ChatRoom.
// The Run method will draw the UI to the terminal in "fullscreen"
// mode. You can quit with Ctrl-C, or by typing "/quit" into the
// chat prompt.
type ChatUI struct {
	app  *tview.Application
	chat *Chat

	msgW    io.Writer
	inputCh chan string
	doneCh  chan struct{}

	ctx context.Context
}

// NewChatUI returns a new ChatUI struct that controls the text UI.
// It won't actually do anything until you call Run().
func NewChatUI(ctx context.Context, chat *Chat) *ChatUI {
	chatUI := new(ChatUI)

	app := tview.NewApplication()

	// make a NewChatUI text view to contain our chat messages
	msgBox := tview.NewTextView()
	msgBox.SetDynamicColors(true)
	msgBox.SetBorder(true)
	msgBox.SetTitle("chat2 example")

	// text views are io.Writers, but they don't automatically refresh.
	// this sets a change handler to force the app to redraw when we get
	// new messages to display.
	msgBox.SetChangedFunc(func() {
		app.Draw()
	})

	// an input field for typing messages into
	inputCh := make(chan string, 32)
	input := tview.NewInputField().
		SetLabel(chat.nick + " > ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorBlack)

	// the done func is called when the user hits enter, or tabs out of the field
	input.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			// we don't want to do anything if they just tabbed away
			return
		}
		line := input.GetText()

		if len(line) == 0 {
			// ignore blank lines
			return
		}

		input.SetText("")

		// bail if requested
		if line == "/quit" {
			app.Stop()
			return
		}

		// add peer
		if strings.HasPrefix(line, "/connect ") {
			peer := strings.TrimPrefix(line, "/connect ")
			go func(peer string) {
				chatUI.displayMessage("Connecting to peer...")
				err := chat.node.DialPeer(peer)
				if err != nil {
					chatUI.displayMessage(err.Error())
				} else {
					chatUI.displayMessage("Peer connected succesfully")
				}
			}(peer)
			return
		}

		// list peers
		if line == "/peers" {
			peers := chat.node.PubSub().ListPeers(string(node.DefaultWakuTopic))
			if len(peers) == 0 {
				chatUI.displayMessage("No peers available")
			}
			for _, p := range peers {
				chatUI.displayMessage("- " + p.Pretty())
			}
			return
		}

		// change nick
		if strings.HasPrefix(line, "/nick ") {
			newNick := strings.TrimSpace(strings.TrimPrefix(line, "/nick "))
			chat.nick = newNick
			input.SetLabel(chat.nick + " > ")
			return
		}

		if line == "/help" {
			chatUI.displayMessage(`
Available commands:
/connect multiaddress - dials a node adding it to the list of connected peers
/peers - list of peers connected to this node
/nick newNick - change the user's nickname
/quit - closes the app
`)
			return
		}

		// send the line onto the input chan and reset the field text
		inputCh <- line
	})

	chatPanel := tview.NewFlex().
		AddItem(msgBox, 0, 1, false)

	// flex is a vertical box with the chatPanel on top and the input field at the bottom.
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(chatPanel, 0, 1, false).
		AddItem(input, 1, 1, true)

	app.SetRoot(flex, true)

	chatUI.app = app
	chatUI.msgW = msgBox
	chatUI.chat = chat
	chatUI.ctx = ctx
	chatUI.inputCh = inputCh
	chatUI.doneCh = make(chan struct{}, 1)

	for _, addr := range chat.node.ListenAddresses() {
		chatUI.displayMessage(fmt.Sprintf("Listening on %s", addr))
	}

	return chatUI
}

// Run starts the chat event loop in the background, then starts
// the event loop for the text UI.
func (ui *ChatUI) Run() error {
	ui.displayMessage("\nWelcome, " + ui.chat.nick)
	ui.displayMessage("type /help to see available commands \n")

	go ui.handleEvents()
	defer ui.end()

	return ui.app.Run()
}

// end signals the event loop to exit gracefully
func (ui *ChatUI) end() {
	ui.doneCh <- struct{}{}
}

// displayChatMessage writes a ChatMessage from the room to the message window,
// with the sender's nick highlighted in green.
func (ui *ChatUI) displayChatMessage(cm *pb.Chat2Message) {
	t := time.Unix(int64(cm.Timestamp), 0)
	prompt := withColor("green", fmt.Sprintf("<%s> %s:", t.Format("Jan 02, 15:04"), cm.Nick))
	fmt.Fprintf(ui.msgW, "%s %s\n", prompt, cm.Payload)
}

// displayMessage writes a blue message to output
func (ui *ChatUI) displayMessage(msg string) {
	fmt.Fprintf(ui.msgW, "%s\n", withColor("grey", msg))
}

// handleEvents runs an event loop that sends user input to the chat room
// and displays messages received from the chat room. It also periodically
// refreshes the list of peers in the UI.
func (ui *ChatUI) handleEvents() {
	for {
		select {
		case input := <-ui.inputCh:
			err := ui.chat.Publish(input)
			if err != nil {
				printErr("publish error: %s", err)
			}

		case m := <-ui.chat.Messages:
			// when we receive a message from the chat room, print it to the message window
			ui.displayChatMessage(m)

		case <-ui.ctx.Done():
			return

		case <-ui.doneCh:
			return
		}
	}
}

// withColor wraps a string with color tags for display in the messages text box.
func withColor(color, msg string) string {
	return fmt.Sprintf("[%s]%s[-]", color, msg)
}
