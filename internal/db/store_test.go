package db

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestStore_ChatsAndMessages(t *testing.T) {
	tempDB := "test_db.sqlite"
	defer os.Remove(tempDB)
	defer os.Remove(tempDB + "-wal")
	defer os.Remove(tempDB + "-shm")

	store, err := NewStore(tempDB)
	if err != nil {
		t.Fatalf("falha ao criar banco de teste: %v", err)
	}
	defer store.Close()

	// 1. Inserir conversa
	now := time.Now().Truncate(time.Second)
	chat := &Chat{
		JID:             "5511999998888@s.whatsapp.net",
		Name:            "Contato Teste",
		LastMessage:     "Olá!",
		LastMessageTime: now,
		IsGroup:         false,
		UnreadCount:     1,
	}
	if err := store.UpsertChat(chat); err != nil {
		t.Fatalf("UpsertChat falhou: %v", err)
	}

	// 2. Recuperar conversas
	chats, err := store.GetChats()
	if err != nil {
		t.Fatalf("GetChats falhou: %v", err)
	}
	if len(chats) != 1 {
		t.Fatalf("esperado 1 chat, obtido %d", len(chats))
	}
	if chats[0].Name != "Contato Teste" {
		t.Errorf("esperado nome 'Contato Teste', obtido '%s'", chats[0].Name)
	}

	// 3. Inserir mensagens
	msg1 := &Message{
		ID:         "MSG-01",
		ChatJID:    chat.JID,
		SenderJID:  chat.JID,
		SenderName: "Contato Teste",
		Text:       "Mensagem 1",
		Timestamp:  now.Add(-1 * time.Minute),
		IsFromMe:   false,
		Status:     "DELIVERED",
	}
	msg2 := &Message{
		ID:         "MSG-02",
		ChatJID:    chat.JID,
		SenderJID:  "eu@s.whatsapp.net",
		SenderName: "Você",
		Text:       "Mensagem 2 de resposta",
		Timestamp:  now,
		IsFromMe:   true,
		Status:     "SENT",
	}

	if err := store.SaveMessage(msg1); err != nil {
		t.Fatalf("SaveMessage 1 falhou: %v", err)
	}
	if err := store.SaveMessage(msg2); err != nil {
		t.Fatalf("SaveMessage 2 falhou: %v", err)
	}

	// 4. Recuperar mensagens em ordem cronológica crescente
	msgs, err := store.GetMessages(chat.JID, 10)
	if err != nil {
		t.Fatalf("GetMessages falhou: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("esperado 2 mensagens, obtido %d", len(msgs))
	}
	if msgs[0].ID != "MSG-01" || msgs[1].ID != "MSG-02" {
		t.Errorf("ordem incorreta de mensagens: %+v", msgs)
	}

	// 5. Resetar contagem de não lidos
	if err := store.ResetUnread(chat.JID); err != nil {
		t.Fatalf("ResetUnread falhou: %v", err)
	}
	chats, _ = store.GetChats()
	if chats[0].UnreadCount != 0 {
		t.Errorf("esperado unread_count = 0, obtido %d", chats[0].UnreadCount)
	}
}

func TestStore_Batches(t *testing.T) {
	tempDB := "test_batch.sqlite"
	defer os.Remove(tempDB)
	defer os.Remove(tempDB + "-wal")
	defer os.Remove(tempDB + "-shm")

	store, err := NewStore(tempDB)
	if err != nil {
		t.Fatalf("falha ao criar banco de teste: %v", err)
	}
	defer store.Close()

	var chats []*Chat
	for i := 0; i < 50; i++ {
		chats = append(chats, &Chat{
			JID:             fmt.Sprintf("chat-%d@s.whatsapp.net", i),
			Name:            fmt.Sprintf("Contato %d", i),
			LastMessage:     "Última msg",
			LastMessageTime: time.Now(),
		})
	}
	if err := store.UpsertChatsBatch(chats); err != nil {
		t.Fatalf("UpsertChatsBatch falhou: %v", err)
	}

	loadedChats, err := store.GetChats()
	if err != nil || len(loadedChats) != 50 {
		t.Fatalf("esperado 50 chats, obtido %d (err: %v)", len(loadedChats), err)
	}

	var msgs []*Message
	for i := 0; i < 100; i++ {
		msgs = append(msgs, &Message{
			ID:        fmt.Sprintf("M-%d", i),
			ChatJID:   "chat-0@s.whatsapp.net",
			SenderJID: "sender@s.whatsapp.net",
			Text:      fmt.Sprintf("Mensagem em lote %d", i),
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			Status:    "DELIVERED",
		})
	}
	if err := store.SaveMessagesBatch(msgs); err != nil {
		t.Fatalf("SaveMessagesBatch falhou: %v", err)
	}

	loadedMsgs, err := store.GetMessages("chat-0@s.whatsapp.net", 35)
	if err != nil || len(loadedMsgs) != 35 {
		t.Fatalf("esperado 35 mensagens carregadas com limite, obtido %d", len(loadedMsgs))
	}
}
