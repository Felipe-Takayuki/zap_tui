package whatsapp

import (
	"zap_tui/internal/db"

	"go.mau.fi/whatsmeow/types"
)

// MsgQR é despachado quando um novo QR Code precisa ser escaneado
type MsgQR struct {
	Code      string
	AsciiView string
}

// MsgConnected é despachado quando o WhatsApp estabelece conexão com sucesso
type MsgConnected struct {
	UserJID  types.JID
	PushName string
}

// MsgDisconnected é despachado quando a conexão é perdida
type MsgDisconnected struct {
	Reason string
}

// MsgLoggedOut é despachado quando a sessão é deslogada
type MsgLoggedOut struct {
	Reason string
}

// MsgNewMessage é despachado quando uma mensagem é recebida ou enviada
type MsgNewMessage struct {
	Message *db.Message
}

// MsgChatsLoaded é despachado quando a lista inicial de conversas é carregada
type MsgChatsLoaded struct {
	Chats []*db.Chat
}

// MsgError é despachado em caso de erro na camada do WhatsApp
type MsgError struct {
	Err error
}

// MsgContactUpdated é despachado quando o nome de um contato é descoberto ou atualizado
type MsgContactUpdated struct {
	JID  string
	Name string
}
