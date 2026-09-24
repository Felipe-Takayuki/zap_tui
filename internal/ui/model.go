package ui

import (
	"fmt"
	"strings"
	"time"

	"zap_tui/internal/db"
	"zap_tui/internal/ui/components"
	"zap_tui/internal/ui/views"
	"zap_tui/internal/whatsapp"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"go.mau.fi/whatsmeow/types"
)

const initialMessageLimit = 35

// ScreenState define a tela atual sendo renderizada
type ScreenState int

const (
	StateLogin ScreenState = iota
	StateChat
)

// MsgSendResult indica o resultado assíncrono do envio de mensagem
type MsgSendResult struct {
	Message *db.Message
	Err     error
}

// MsgLoadedMessages transporta mensagens lidas do SQLite em segundo plano
type MsgLoadedMessages struct {
	ChatJID  string
	Messages []*db.Message
}

// Model é o modelo raiz do Bubble Tea
type Model struct {
	state   ScreenState
	focus   views.FocusArea
	status  string
	qrAscii string

	userName string
	userJID  string

	chats           []*db.Chat
	selectedChatIdx int
	activeChat      *db.Chat
	messages        map[string][]*db.Message
	renderedCache   map[string]string

	vp    viewport.Model
	input textinput.Model

	width  int
	height int

	client *whatsapp.Client
	store  *db.Store
}

// NewModel inicializa o modelo da interface
func NewModel(client *whatsapp.Client, store *db.Store) *Model {
	ti := textinput.New()
	ti.Placeholder = "Digite uma mensagem ou /to <número> <mensagem>..."
	ti.CharLimit = 2048
	ti.Prompt = "💬 "

	vp := viewport.New(60, 20)

	initialState := StateChat
	status := "Verificando sessão..."
	if !client.HasSavedSession() {
		initialState = StateLogin
		status = "Aguardando geração do QR Code..."
	}

	m := &Model{
		state:         initialState,
		focus:         views.FocusInput,
		status:        status,
		messages:      make(map[string][]*db.Message),
		renderedCache: make(map[string]string),
		vp:            vp,
		input:         ti,
		client:        client,
		store:         store,
		width:         80,
		height:        24,
	}

	ti.Focus()
	return m
}

// Init inicializa os comandos iniciais do Bubble Tea
func (m *Model) Init() tea.Cmd {
	chats, err := m.store.GetChats()
	if err == nil && len(chats) > 0 {
		m.chats = chats
		m.selectedChatIdx = 0
		return tea.Batch(m.selectChat(0, false), textinput.Blink)
	}

	return textinput.Blink
}

// Update processa todos os eventos e mensagens no loop do Bubble Tea
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		clear(m.renderedCache) // Limpa o cache ao redimensionar
		m.updateDimensions()

	case whatsapp.MsgQR:
		m.state = StateLogin
		m.qrAscii = msg.AsciiView
		m.status = "Aguardando leitura do QR Code"

	case whatsapp.MsgConnected:
		m.state = StateChat
		m.status = "Conectado"
		m.userJID = msg.UserJID.User
		m.userName = msg.PushName

	case whatsapp.MsgDisconnected:
		m.status = "Desconectado: " + msg.Reason

	case whatsapp.MsgLoggedOut:
		m.state = StateLogin
		m.status = "Sessão desconectada: " + msg.Reason
		m.qrAscii = ""

	case whatsapp.MsgChatsLoaded:
		m.chats = msg.Chats
		if m.activeChat == nil && len(m.chats) > 0 {
			cmds = append(cmds, m.selectChat(0, false))
		}

	case MsgLoadedMessages:
		m.messages[msg.ChatJID] = msg.Messages
		delete(m.renderedCache, msg.ChatJID)
		if m.activeChat != nil && m.activeChat.JID == msg.ChatJID {
			m.refreshViewport()
			m.vp.GotoBottom()
		}

	case whatsapp.MsgNewMessage:
		m.handleIncomingMessage(msg.Message)

	case MsgSendResult:
		if msg.Err != nil {
			m.status = fmt.Sprintf("Erro ao enviar: %v", msg.Err)
		} else {
			m.handleOutgoingMessage(msg.Message)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "tab":
			m.cycleFocus(1)
			return m, nil

		case "shift+tab":
			m.cycleFocus(-1)
			return m, nil

		case "esc":
			m.focus = views.FocusList
			m.input.Blur()
			return m, nil
		}

		switch m.focus {
		case views.FocusList:
			switch msg.String() {
			case "up", "k":
				if m.selectedChatIdx > 0 {
					m.selectedChatIdx--
					loadCmd := m.selectChat(m.selectedChatIdx, false)
					if loadCmd != nil {
						cmds = append(cmds, loadCmd)
					}
				}
			case "down", "j":
				if m.selectedChatIdx < len(m.chats)-1 {
					m.selectedChatIdx++
					loadCmd := m.selectChat(m.selectedChatIdx, false)
					if loadCmd != nil {
						cmds = append(cmds, loadCmd)
					}
				}
			case "enter":
				loadCmd := m.selectChat(m.selectedChatIdx, true)
				if loadCmd != nil {
					cmds = append(cmds, loadCmd)
				}
				m.focus = views.FocusInput
				cmds = append(cmds, m.input.Focus())
			}

		case views.FocusMessages:
			m.vp, cmd = m.vp.Update(msg)
			cmds = append(cmds, cmd)

		case views.FocusInput:
			if msg.String() == "enter" {
				val := strings.TrimSpace(m.input.Value())
				if val != "" {
					sendCmd := m.handleInputSubmit(val)
					m.input.Reset()
					if sendCmd != nil {
						cmds = append(cmds, sendCmd)
					}
				}
			} else {
				m.input, cmd = m.input.Update(msg)
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// View renderiza a interface no terminal
func (m *Model) View() string {
	if m.state == StateLogin {
		return views.RenderQRView(m.width, m.height, m.qrAscii, m.status)
	}

	return views.RenderChatView(
		m.width,
		m.height,
		m.chats,
		m.selectedChatIdx,
		m.activeChat,
		m.vp,
		m.input,
		m.focus,
		m.status,
		m.userName,
		m.userJID,
	)
}

func (m *Model) cycleFocus(delta int) {
	if delta > 0 {
		switch m.focus {
		case views.FocusList:
			m.focus = views.FocusInput
			m.input.Focus()
		case views.FocusInput:
			m.focus = views.FocusMessages
			m.input.Blur()
		case views.FocusMessages:
			m.focus = views.FocusList
			m.input.Blur()
		}
	} else {
		switch m.focus {
		case views.FocusList:
			m.focus = views.FocusMessages
			m.input.Blur()
		case views.FocusMessages:
			m.focus = views.FocusInput
			m.input.Focus()
		case views.FocusInput:
			m.focus = views.FocusList
			m.input.Blur()
		}
	}
}

func (m *Model) updateDimensions() {
	headerHeight := 2
	footerHeight := 2
	bodyHeight := m.height - headerHeight - footerHeight
	if bodyHeight < 6 {
		bodyHeight = 6
	}

	leftWidth := m.width * 28 / 100
	if leftWidth < 26 {
		leftWidth = 26
	}
	if leftWidth > 40 {
		leftWidth = 40
	}
	rightWidth := m.width - leftWidth

	vpHeight := bodyHeight - 6
	if vpHeight < 2 {
		vpHeight = 2
	}
	vpWidth := rightWidth - 4
	if vpWidth < 10 {
		vpWidth = 10
	}

	m.vp.Width = vpWidth
	m.vp.Height = vpHeight
	m.input.Width = vpWidth - 4

	m.refreshViewport()
}

// selectChat seleciona uma conversa da lista e carrega as mensagens de forma assíncrona se necessário
func (m *Model) selectChat(idx int, enterChat bool) tea.Cmd {
	if idx < 0 || idx >= len(m.chats) {
		return nil
	}
	m.selectedChatIdx = idx
	m.activeChat = m.chats[idx]
	chatJID := m.activeChat.JID

	if enterChat {
		m.client.SetActiveChat(chatJID)
		m.activeChat.UnreadCount = 0
		go func(target string) {
			_ = m.store.ResetUnread(target)
		}(chatJID)
	}

	// Se as mensagens já estiverem carregadas em memória, renderiza instantaneamente
	if _, ok := m.messages[chatJID]; ok {
		m.refreshViewport()
		m.vp.GotoBottom()
		return nil
	}

	// Se não estiver em memória, carrega em background sem travar a navegação
	m.vp.SetContent("Carregando mensagens...")
	return m.loadMessagesCmd(chatJID, initialMessageLimit)
}

func (m *Model) loadMessagesCmd(chatJID string, limit int) tea.Cmd {
	return func() tea.Msg {
		msgs, err := m.store.GetMessages(chatJID, limit)
		if err != nil {
			return MsgLoadedMessages{ChatJID: chatJID, Messages: nil}
		}
		return MsgLoadedMessages{ChatJID: chatJID, Messages: msgs}
	}
}

func (m *Model) refreshViewport() {
	if m.activeChat == nil {
		m.vp.SetContent("Selecione uma conversa na barra lateral ou use /to <número> <mensagem>.")
		return
	}

	cacheKey := fmt.Sprintf("%s:%d", m.activeChat.JID, m.vp.Width)
	if cached, ok := m.renderedCache[cacheKey]; ok {
		m.vp.SetContent(cached)
		return
	}

	msgs := m.messages[m.activeChat.JID]
	rendered := components.RenderMessageHistory(msgs, m.vp.Width)
	m.renderedCache[cacheKey] = rendered
	m.vp.SetContent(rendered)
}

func (m *Model) handleIncomingMessage(msg *db.Message) {
	m.messages[msg.ChatJID] = append(m.messages[msg.ChatJID], msg)
	delete(m.renderedCache, fmt.Sprintf("%s:%d", msg.ChatJID, m.vp.Width))

	if m.activeChat != nil && m.activeChat.JID == msg.ChatJID {
		m.refreshViewport()
		m.vp.GotoBottom()
	}

	m.updateChatInMemory(msg.ChatJID, msg.SenderName, msg.Text, msg.Timestamp, !msg.IsFromMe)
}

func (m *Model) handleOutgoingMessage(msg *db.Message) {
	m.messages[msg.ChatJID] = append(m.messages[msg.ChatJID], msg)
	delete(m.renderedCache, fmt.Sprintf("%s:%d", msg.ChatJID, m.vp.Width))

	if m.activeChat != nil && m.activeChat.JID == msg.ChatJID {
		m.refreshViewport()
		m.vp.GotoBottom()
	}

	m.updateChatInMemory(msg.ChatJID, "Você", msg.Text, msg.Timestamp, false)
}

// updateChatInMemory atualiza a barra lateral instantaneamente na memória sem fazer queries pesadas no SQLite
func (m *Model) updateChatInMemory(chatJID, name, text string, t time.Time, isIncoming bool) {
	for i, c := range m.chats {
		if c.JID == chatJID {
			c.LastMessage = text
			c.LastMessageTime = t
			if isIncoming && (m.activeChat == nil || m.activeChat.JID != chatJID) {
				c.UnreadCount++
			}

			// Move conversa para o topo da lista
			if i > 0 {
				target := m.chats[i]
				copy(m.chats[1:i+1], m.chats[0:i])
				m.chats[0] = target
				if m.selectedChatIdx == i {
					m.selectedChatIdx = 0
				} else if m.selectedChatIdx < i {
					m.selectedChatIdx++
				}
			}
			return
		}
	}

	// Chat novo que ainda não estava na lista
	newChat := &db.Chat{
		JID:             chatJID,
		Name:            name,
		LastMessage:     text,
		LastMessageTime: t,
	}
	if isIncoming {
		newChat.UnreadCount = 1
	}
	m.chats = append([]*db.Chat{newChat}, m.chats...)
	m.selectedChatIdx++
}

// handleInputSubmit processa envio normal ou comandos via slash (ex: /to, /new)
func (m *Model) handleInputSubmit(inputVal string) tea.Cmd {
	if strings.HasPrefix(inputVal, "/to ") {
		parts := strings.SplitN(strings.TrimPrefix(inputVal, "/to "), " ", 2)
		if len(parts) < 2 {
			m.status = "Uso: /to <número> <mensagem>"
			return nil
		}
		targetPhone := parts[0]
		text := parts[1]

		targetJID, err := whatsapp.ParseTargetJID(targetPhone)
		if err != nil {
			m.status = "Número inválido: " + err.Error()
			return nil
		}

		return m.sendMessageCmd(targetJID, text)
	}

	if strings.HasPrefix(inputVal, "/new ") {
		phone := strings.TrimSpace(strings.TrimPrefix(inputVal, "/new "))
		targetJID, err := whatsapp.ParseTargetJID(phone)
		if err != nil {
			m.status = "Número inválido: " + err.Error()
			return nil
		}

		newChat := &db.Chat{
			JID:             targetJID.String(),
			Name:            targetJID.User,
			LastMessage:     "",
			LastMessageTime: time.Now(),
		}
		_ = m.store.UpsertChat(newChat)
		m.chats = append([]*db.Chat{newChat}, m.chats...)
		m.selectedChatIdx = 0
		return m.selectChat(0, true)
	}

	if m.activeChat == nil {
		m.status = "Nenhuma conversa selecionada. Use /to <número> <mensagem>."
		return nil
	}

	targetJID, err := types.ParseJID(m.activeChat.JID)
	if err != nil {
		m.status = "JID do chat atual inválido"
		return nil
	}

	return m.sendMessageCmd(targetJID, inputVal)
}

func (m *Model) sendMessageCmd(targetJID types.JID, text string) tea.Cmd {
	return func() tea.Msg {
		msg, err := m.client.SendMessage(targetJID, text)
		return MsgSendResult{
			Message: msg,
			Err:     err,
		}
	}
}
