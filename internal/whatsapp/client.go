package whatsapp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"zap_tui/internal/db"

	_ "modernc.org/sqlite"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Client encapsula a conexão com o WhatsApp e a sincronização local
type Client struct {
	cli           *whatsmeow.Client
	container     *sqlstore.Container
	store         *db.Store
	dispatcher    func(any)
	loggerCleanup func()

	mu            sync.RWMutex
	activeChatJID string
	connected     bool
}

// NewClient cria e configura uma instância do cliente whatsmeow integrada ao SQLite
func NewClient(dbPath, logFile string, storeDB *db.Store, dispatcher func(any)) (*Client, error) {
	logger, cleanup, err := NewLogger(logFile, "WhatsApp")
	if err != nil {
		return nil, fmt.Errorf("falha ao inicializar logger: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)
	container, err := sqlstore.New(context.Background(), "sqlite", dsn, logger)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("falha ao inicializar armazenamento whatsmeow: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		_ = container.Close()
		cleanup()
		return nil, fmt.Errorf("falha ao buscar dispositivo no banco: %w", err)
	}

	if deviceStore == nil {
		deviceStore = container.NewDevice()
	}

	cli := whatsmeow.NewClient(deviceStore, logger)

	c := &Client{
		cli:           cli,
		container:     container,
		store:         storeDB,
		dispatcher:    dispatcher,
		loggerCleanup: cleanup,
	}

	cli.AddEventHandler(c.handleEvent)

	return c, nil
}

// HasSavedSession verifica se já existe uma sessão pareada previamente
func (c *Client) HasSavedSession() bool {
	return c.cli.Store.ID != nil
}

// SetActiveChat define qual chat está visualizado no momento para evitar notificações indevidas
func (c *Client) SetActiveChat(chatJID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.activeChatJID = chatJID
	if c.store != nil && chatJID != "" {
		_ = c.store.ResetUnread(chatJID)
	}
}

// Connect inicia a conexão com o WhatsApp. Se não houver sessão, escuta o canal de QR Code
func (c *Client) Connect(ctx context.Context) error {
	if c.cli.Store.ID == nil {
		// Não pareado: precisa exibir o QR Code
		qrChan, err := c.cli.GetQRChannel(ctx)
		if err != nil {
			return fmt.Errorf("falha ao obter canal de qr code: %w", err)
		}

		err = c.cli.Connect()
		if err != nil {
			return fmt.Errorf("falha ao conectar cliente whatsapp: %w", err)
		}

		go func() {
			for item := range qrChan {
				switch item.Event {
				case "code":
					ascii := GenerateQRAscii(item.Code)
					c.dispatch(MsgQR{
						Code:      item.Code,
						AsciiView: ascii,
					})
				case "success":
					// Sucesso tratado automaticamente no evento Connected
				case "timeout":
					c.dispatch(MsgError{Err: fmt.Errorf("tempo limite do QR code expirado")})
				case "error":
					c.dispatch(MsgError{Err: item.Error})
				}
			}
		}()
	} else {
		// Já pareado: conectar diretamente
		if err := c.cli.Connect(); err != nil {
			return fmt.Errorf("falha ao reconectar sessão salva: %w", err)
		}
	}

	return nil
}

func (c *Client) dispatch(msg any) {
	if c.dispatcher != nil {
		c.dispatcher(msg)
	}
}

// handleEvent intercepta todos os eventos de rede do whatsmeow
func (c *Client) handleEvent(rawEvt any) {
	switch evt := rawEvt.(type) {
	case *events.Connected:
		c.mu.Lock()
		c.connected = true
		c.mu.Unlock()

		userJID := types.EmptyJID
		if c.cli.Store.ID != nil {
			userJID = *c.cli.Store.ID
		}

		c.dispatch(MsgConnected{
			UserJID:  userJID,
			PushName: c.cli.Store.PushName,
		})

		// Sincronizar contatos e grupos iniciais em background
		go c.syncInitialData()

	case *events.PushName:
		if evt.NewPushName != "" {
			_, _, _ = c.cli.Store.Contacts.PutPushName(context.Background(), evt.JID, evt.NewPushName)
			_ = c.store.UpdateChatName(evt.JID.String(), evt.NewPushName)
			c.dispatch(MsgContactUpdated{JID: evt.JID.String(), Name: evt.NewPushName})
		}

	case *events.Contact:
		if evt.Action != nil {
			name := evt.Action.GetFullName()
			if name == "" {
				name = evt.Action.GetFirstName()
			}
			if name != "" {
				_ = c.cli.Store.Contacts.PutContactName(context.Background(), evt.JID, name, evt.Action.GetFirstName())
				_ = c.store.UpdateChatName(evt.JID.String(), name)
				c.dispatch(MsgContactUpdated{JID: evt.JID.String(), Name: name})
			}
		}

	case *events.Disconnected:
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		c.dispatch(MsgDisconnected{Reason: "Conexão perdida com os servidores do WhatsApp"})

	case *events.LoggedOut:
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		c.dispatch(MsgLoggedOut{Reason: evt.PermanentDisconnectDescription()})

	case *events.HistorySync:
		go c.handleHistorySync(evt.Data)

	case *events.Message:
		c.handleIncomingMessage(evt)
	}
}

// syncInitialData carrega contatos e grupos salvos e emite a lista para a TUI
func (c *Client) syncInitialData() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Inicia a sincronização de patches de AppState (onde a agenda e contatos são sincronizados)
	go func() {
		ctxSync, cancelSync := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelSync()
		for _, patch := range appstate.AllPatchNames {
			_ = c.cli.FetchAppState(ctxSync, patch, false, false)
		}
	}()

	var chatsBatch []*db.Chat

	// 1. Carregar grupos conhecidos
	groups, err := c.cli.GetJoinedGroups(ctx)
	if err == nil {
		for _, g := range groups {
			chatsBatch = append(chatsBatch, &db.Chat{
				JID:             g.JID.String(),
				Name:            g.Name,
				LastMessageTime: g.GroupCreated,
				IsGroup:         true,
			})
		}
	}

	// 2. Carregar contatos já conhecidos no store do whatsmeow
	contacts, err := c.cli.Store.Contacts.GetAllContacts(ctx)
	if err == nil {
		for jid, info := range contacts {
			name := info.FullName
			if name == "" {
				name = info.BusinessName
			}
			if name == "" {
				name = info.PushName
			}
			if name == "" {
				name = info.FirstName
			}
			if name == "" {
				name = jid.User
			}
			chatsBatch = append(chatsBatch, &db.Chat{
				JID:             jid.String(),
				Name:            name,
				LastMessageTime: time.Time{},
				IsGroup:         jid.Server == types.GroupServer,
			})
		}
	}

	if len(chatsBatch) > 0 {
		_ = c.store.UpsertChatsBatch(chatsBatch)
	}

	// 3. Emitir lista atualizada de conversas locais
	chats, err := c.store.GetChats()
	if err == nil && len(chats) > 0 {
		c.dispatch(MsgChatsLoaded{Chats: chats})
	}
}

// handleHistorySync processa blocos de sincronização de histórico do WhatsApp
func (c *Client) handleHistorySync(sync *waHistorySync.HistorySync) {
	if sync == nil {
		return
	}

	// 1. Processar nomes da agenda de contatos sincronizados pelo celular
	for _, ic := range sync.GetInlineContacts() {
		pnJIDStr := ic.GetPnJID()
		if pnJIDStr == "" {
			continue
		}
		jid, err := types.ParseJID(pnJIDStr)
		if err != nil {
			continue
		}
		name := ic.GetFullName()
		if name == "" {
			name = ic.GetFirstName()
		}
		if name != "" {
			_ = c.cli.Store.Contacts.PutContactName(context.Background(), jid, name, ic.GetFirstName())
			_ = c.store.UpdateChatName(jid.String(), name)
			c.dispatch(MsgContactUpdated{JID: jid.String(), Name: name})
		}
	}

	// 2. Processar nomes de perfil (Pushnames) recebidos
	for _, pn := range sync.GetPushnames() {
		idStr := pn.GetID()
		pushName := pn.GetPushname()
		if idStr == "" || pushName == "" {
			continue
		}
		jid, err := types.ParseJID(idStr)
		if err != nil {
			continue
		}
		_, _, _ = c.cli.Store.Contacts.PutPushName(context.Background(), jid, pushName)
		_ = c.store.UpdateChatName(jid.String(), pushName)
		c.dispatch(MsgContactUpdated{JID: jid.String(), Name: pushName})
	}

	var msgsBatch []*db.Message
	var chatsBatch []*db.Chat

	for _, conv := range sync.GetConversations() {
		chatJIDStr := conv.GetID()
		chatJID, err := types.ParseJID(chatJIDStr)
		if err != nil {
			continue
		}

		chatName := conv.GetName()
		if chatName == "" {
			chatName = c.ResolveContactName(chatJID)
		}

		var lastMsgText string
		var lastMsgTime time.Time

		for _, historyMsg := range conv.GetMessages() {
			webMsg := historyMsg.GetMessage()
			if webMsg == nil {
				continue
			}

			evtMsg, err := c.cli.ParseWebMessage(chatJID, webMsg)
			if err != nil || evtMsg == nil {
				continue
			}

			text := extractText(evtMsg.Message)
			if text == "" {
				continue
			}

			ts := evtMsg.Info.Timestamp
			if ts.After(lastMsgTime) {
				lastMsgTime = ts
				lastMsgText = text
			}

			senderName := evtMsg.Info.PushName
			if senderName == "" {
				senderName = c.ResolveContactName(evtMsg.Info.Sender)
			}
			if evtMsg.Info.IsFromMe {
				senderName = "Você"
			}

			msgsBatch = append(msgsBatch, &db.Message{
				ID:         evtMsg.Info.ID,
				ChatJID:    chatJID.String(),
				SenderJID:  evtMsg.Info.Sender.String(),
				SenderName: senderName,
				Text:       text,
				Timestamp:  ts,
				IsFromMe:   evtMsg.Info.IsFromMe,
				Status:     "DELIVERED",
			})
		}

		if lastMsgTime.IsZero() {
			lastMsgTime = time.Now()
		}

		chatsBatch = append(chatsBatch, &db.Chat{
			JID:             chatJID.String(),
			Name:            chatName,
			LastMessage:     lastMsgText,
			LastMessageTime: lastMsgTime,
			IsGroup:         chatJID.Server == types.GroupServer,
		})
	}

	// Grava mensagens e conversas em batch único
	if len(msgsBatch) > 0 {
		_ = c.store.SaveMessagesBatch(msgsBatch)
	}
	if len(chatsBatch) > 0 {
		_ = c.store.UpsertChatsBatch(chatsBatch)
	}

	// Recarrega conversas para a UI
	chats, err := c.store.GetChats()
	if err == nil {
		c.dispatch(MsgChatsLoaded{Chats: chats})
	}
}

// handleIncomingMessage trata mensagens em tempo real recebidas de outros ou de seus próprios dispositivos
func (c *Client) handleIncomingMessage(evt *events.Message) {
	text := extractText(evt.Message)
	if text == "" {
		return
	}

	chatJID := evt.Info.Chat.String()
	senderJID := evt.Info.Sender.String()

	senderName := evt.Info.PushName
	if senderName != "" {
		_, _, _ = c.cli.Store.Contacts.PutPushName(context.Background(), evt.Info.Sender, senderName)
		if !evt.Info.IsGroup && !evt.Info.IsFromMe {
			_ = c.store.UpdateChatName(chatJID, senderName)
			c.dispatch(MsgContactUpdated{JID: chatJID, Name: senderName})
		}
	} else {
		senderName = c.ResolveContactName(evt.Info.Sender)
	}
	if evt.Info.IsFromMe {
		senderName = "Você"
	}

	c.mu.RLock()
	isCurrentChat := (c.activeChatJID == chatJID)
	c.mu.RUnlock()

	msg := &db.Message{
		ID:         evt.Info.ID,
		ChatJID:    chatJID,
		SenderJID:  senderJID,
		SenderName: senderName,
		Text:       text,
		Timestamp:  evt.Info.Timestamp,
		IsFromMe:   evt.Info.IsFromMe,
		Status:     "DELIVERED",
	}

	_ = c.store.SaveMessage(msg)
	_ = c.store.UpdateChatLastMessage(chatJID, senderName, text, evt.Info.Timestamp, !evt.Info.IsFromMe, isCurrentChat)

	c.dispatch(MsgNewMessage{Message: msg})
}

// SendMessage envia uma mensagem de texto pelo whatsmeow e salva no banco local
func (c *Client) SendMessage(to types.JID, text string) (*db.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	waMsg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	resp, err := c.cli.SendMessage(ctx, to, waMsg)
	if err != nil {
		return nil, fmt.Errorf("erro ao enviar mensagem via whatsmeow: %w", err)
	}

	myJID := ""
	if c.cli.Store.ID != nil {
		myJID = c.cli.Store.ID.String()
	}

	msg := &db.Message{
		ID:         resp.ID,
		ChatJID:    to.String(),
		SenderJID:  myJID,
		SenderName: "Você",
		Text:       text,
		Timestamp:  resp.Timestamp,
		IsFromMe:   true,
		Status:     "SENT",
	}

	_ = c.store.SaveMessage(msg)
	_ = c.store.UpdateChatLastMessage(to.String(), to.User, text, resp.Timestamp, false, true)

	return msg, nil
}

// Disconnect desconecta a sessão de forma segura
func (c *Client) Disconnect() {
	c.cli.Disconnect()
}

// Close desconecta o cliente e fecha o banco de dados
func (c *Client) Close() {
	c.cli.Disconnect()
	if c.container != nil {
		_ = c.container.Close()
	}
	if c.loggerCleanup != nil {
		c.loggerCleanup()
	}
}

// ParseTargetJID converte uma entrada do usuário (número de telefone ou JID) em um types.JID válido
func ParseTargetJID(input string) (types.JID, error) {
	input = strings.TrimSpace(input)
	if strings.Contains(input, "@") {
		return types.ParseJID(input)
	}

	// Remove caracteres não numéricos comuns (+, -, espaço, parênteses)
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, input)

	if len(cleaned) < 8 {
		return types.EmptyJID, fmt.Errorf("número de telefone inválido: %s", input)
	}

	return types.NewJID(cleaned, types.DefaultUserServer), nil
}

func extractText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if text := msg.GetConversation(); text != "" {
		return text
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil && ext.GetText() != "" {
		return ext.GetText()
	}
	if img := msg.GetImageMessage(); img != nil {
		if caption := img.GetCaption(); caption != "" {
			return "📷 " + caption
		}
		return "📷 [Foto]"
	}
	if vid := msg.GetVideoMessage(); vid != nil {
		if caption := vid.GetCaption(); caption != "" {
			return "🎥 " + caption
		}
		return "🎥 [Vídeo]"
	}
	if doc := msg.GetDocumentMessage(); doc != nil {
		if title := doc.GetTitle(); title != "" {
			return "📄 " + title
		}
		return "📄 [Documento]"
	}
	if aud := msg.GetAudioMessage(); aud != nil {
		return "🎵 [Áudio]"
	}
	if stk := msg.GetStickerMessage(); stk != nil {
		return "🏷️ [Figurinha]"
	}
	if loc := msg.GetLocationMessage(); loc != nil {
		return "📍 [Localização]"
	}
	if contact := msg.GetContactMessage(); contact != nil {
		return "👤 [Contato: " + contact.GetDisplayName() + "]"
	}
	return ""
}

// ResolveContactName obtém o melhor nome para um contato (Agenda > Business > PushName > Primeiro Nome > Telefone)
func (c *Client) ResolveContactName(jid types.JID) string {
	if jid.Server == types.GroupServer {
		return jid.User
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if c.cli != nil && c.cli.Store != nil && c.cli.Store.Contacts != nil {
		contact, err := c.cli.Store.Contacts.GetContact(ctx, jid)
		if err == nil && contact.Found {
			if contact.FullName != "" {
				return contact.FullName
			}
			if contact.BusinessName != "" {
				return contact.BusinessName
			}
			if contact.PushName != "" {
				return contact.PushName
			}
			if contact.FirstName != "" {
				return contact.FirstName
			}
		}
	}

	if c.store != nil {
		chat, err := c.store.GetChat(jid.String())
		if err == nil && chat != nil && chat.Name != "" && chat.Name != jid.User {
			return chat.Name
		}
	}

	return jid.User
}
