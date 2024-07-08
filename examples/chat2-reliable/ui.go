package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
	"github.com/waku-org/go-waku/waku/v2/utils"
)

const viewportMargin = 6

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}().Render

	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render
	infoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Render
	senderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Render
)

type errMsg error

type sending bool

type quit bool

type MessageType int

const (
	ChatMessageType MessageType = iota
	InfoMessageType
	ErrorMessageType
)

type message struct {
	mType   MessageType
	err     error
	author  string
	clock   time.Time
	content string
}

type UI struct {
	ready bool
	err   error

	quitChan  chan struct{}
	readyChan chan<- struct{}
	inputChan chan<- string

	messageChan chan message
	messages    []message

	isSendingChan chan sending
	isSending     bool

	width  int
	height int

	viewport viewport.Model
	textarea textarea.Model

	spinner spinner.Model
}

func NewUIModel(readyChan chan<- struct{}, inputChan chan<- string) UI {
	width, height := GetTerminalDimensions()

	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	ta.Prompt = "┃ "
	ta.CharLimit = 2000

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.SetHeight(3)
	ta.SetWidth(width)
	ta.ShowLineNumbers = false

	ta.KeyMap.InsertNewline.SetEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Jump
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := UI{
		messageChan:   make(chan message, 100),
		isSendingChan: make(chan sending, 100),
		quitChan:      make(chan struct{}),
		readyChan:     readyChan,
		inputChan:     inputChan,
		width:         width,
		height:        height,
		textarea:      ta,
		spinner:       s,
		err:           nil,
	}

	return m
}

func (m UI) Init() tea.Cmd {
	return tea.Batch(
		recvQuitSignal(m.quitChan),
		recvMessages(m.messageChan),
		recvSendingState(m.isSendingChan),
		textarea.Blink,
		spinner.Tick,
	)
}

func (m UI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	var cmdToReturn []tea.Cmd = []tea.Cmd{tiCmd, vpCmd}

	headerHeight := lipgloss.Height(m.headerView())

	printMessages := false

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-viewportMargin)
			m.viewport.SetContent("")
			m.viewport.YPosition = headerHeight + 1
			m.viewport.KeyMap = DefaultKeyMap()
			m.ready = true

			close(m.readyChan)
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - viewportMargin
		}

		printMessages = true

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			line := m.textarea.Value()
			if len(line) != 0 {
				m.inputChan <- line
				m.textarea.Reset()
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil

	case message:
		m.messages = append(m.messages, msg)
		printMessages = true
		cmdToReturn = append(cmdToReturn, recvMessages(m.messageChan))

	case quit:
		fmt.Println("Bye!")
		return m, tea.Quit

	case sending:
		m.isSending = bool(msg)
		cmdToReturn = append(cmdToReturn, recvSendingState(m.isSendingChan))

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if printMessages {
		var sb strings.Builder
		for i, msg := range m.messages {
			line := ""

			switch msg.mType {
			case ChatMessageType:
				line += m.breaklineIfNeeded(i, ChatMessageType)
				msgLine := "[" + msg.clock.Format("Jan 02 15:04") + " " + senderStyle(msg.author) + "] "
				msgLine += msg.content
				line += wordwrap.String(line+msgLine, m.width-10)
			case ErrorMessageType:
				line += m.breaklineIfNeeded(i, ErrorMessageType)
				line += wordwrap.String(errorStyle("ERROR:")+" "+msg.err.Error(), m.width-10)
				utils.Logger().Error(msg.content)
			case InfoMessageType:
				line += m.breaklineIfNeeded(i, InfoMessageType)
				line += wordwrap.String(infoStyle("INFO:")+" "+msg.content, m.width-10)
				utils.Logger().Info(msg.content)
			}

			sb.WriteString(line + "\n")

		}

		m.viewport.SetContent(sb.String())
		m.viewport.GotoBottom()
	}

	return m, tea.Batch(cmdToReturn...)
}

func (m UI) breaklineIfNeeded(i int, mt MessageType) string {
	result := ""
	if i > 0 {
		if (mt == ChatMessageType && m.messages[i-1].mType != ChatMessageType) || (mt != ChatMessageType && m.messages[i-1].mType == ChatMessageType) {
			result += "\n"
		}
	}
	return result
}

func (m UI) headerView() string {
	title := titleStyle("Chat2 •")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)-4))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m UI) View() string {
	spinnerStr := ""
	inputStr := ""
	if m.isSending {
		spinnerStr = m.spinner.View() + " Sending message..."
	} else {
		inputStr = m.textarea.View()
	}

	return appStyle.Render(fmt.Sprintf(
		"%s\n%s\n%s%s\n",
		m.headerView(),
		m.viewport.View(),
		inputStr,
		spinnerStr,
	),
	)
}

func recvMessages(sub chan message) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func recvSendingState(sub chan sending) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func recvQuitSignal(q chan struct{}) tea.Cmd {
	return func() tea.Msg {
		<-q
		return quit(true)
	}
}

func (m UI) Quit() {
	m.quitChan <- struct{}{}
}

func (m UI) SetSending(isSending bool) {
	m.isSendingChan <- sending(isSending)
}

func (m UI) ErrorMessage(err error) {
	m.messageChan <- message{mType: ErrorMessageType, err: err}
}

func (m UI) InfoMessage(text string) {
	m.messageChan <- message{mType: InfoMessageType, content: text}
}

func (m UI) ChatMessage(clock int64, author string, text string) {
	m.messageChan <- message{mType: ChatMessageType, author: author, content: text, clock: time.Unix(clock, 0)}
}

// DefaultKeyMap returns a set of pager-like default keybindings.
func DefaultKeyMap() viewport.KeyMap {
	return viewport.KeyMap{
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "½ page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "½ page down"),
		),
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "down"),
		),
	}
}
