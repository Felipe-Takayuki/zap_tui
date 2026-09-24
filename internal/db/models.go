package db

import "time"

// Chat representa um resumo de conversa armazenado localmente
type Chat struct {
	JID             string    `json:"jid"`
	Name            string    `json:"name"`
	LastMessage     string    `json:"last_message"`
	LastMessageTime time.Time `json:"last_message_time"`
	IsGroup         bool      `json:"is_group"`
	UnreadCount     int       `json:"unread_count"`
}

// Message representa uma mensagem de texto armazenada localmente
type Message struct {
	ID         string    `json:"id"`
	ChatJID    string    `json:"chat_jid"`
	SenderJID  string    `json:"sender_jid"`
	SenderName string    `json:"sender_name"`
	Text       string    `json:"text"`
	Timestamp  time.Time `json:"timestamp"`
	IsFromMe   bool      `json:"is_from_me"`
	Status     string    `json:"status"` // SENT, DELIVERED, READ, ERROR
}
