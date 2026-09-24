package db

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Store gerencia o banco de dados SQLite local de mensagens e conversas
type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewStore inicializa o banco SQLite com suporte a WAL e migra as tabelas locais
func NewStore(dbPath string) (*Store, error) {
	connStr := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)&_pragma=cache_size(-64000)&_pragma=busy_timeout(10000)", dbPath)
	db, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir banco sqlite local: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("falha na migração do banco sqlite: %w", err)
	}

	return store, nil
}

func (s *Store) migrate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	CREATE TABLE IF NOT EXISTS local_chats (
		jid TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		last_message TEXT,
		last_message_time DATETIME NOT NULL,
		is_group BOOLEAN NOT NULL DEFAULT 0,
		unread_count INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_chats_time ON local_chats(last_message_time DESC);

	CREATE TABLE IF NOT EXISTS local_messages (
		id TEXT PRIMARY KEY,
		chat_jid TEXT NOT NULL,
		sender_jid TEXT NOT NULL,
		sender_name TEXT,
		text TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		is_from_me BOOLEAN NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'SENT'
	);
	CREATE INDEX IF NOT EXISTS idx_messages_chat_ts ON local_messages(chat_jid, timestamp);
	`
	_, err := s.db.Exec(query)
	return err
}

// GetChats retorna todas as conversas ordenadas pela data da última mensagem
func (s *Store) GetChats() ([]*Chat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT jid, name, COALESCE(last_message, ''), last_message_time, is_group, unread_count
		FROM local_chats
		ORDER BY last_message_time DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []*Chat
	for rows.Next() {
		var c Chat
		if err := rows.Scan(&c.JID, &c.Name, &c.LastMessage, &c.LastMessageTime, &c.IsGroup, &c.UnreadCount); err != nil {
			return nil, err
		}
		chats = append(chats, &c)
	}
	return chats, nil
}

// UpsertChat insere ou atualiza informações básicas de uma conversa
func (s *Store) UpsertChat(chat *Chat) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	INSERT INTO local_chats (jid, name, last_message, last_message_time, is_group, unread_count)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(jid) DO UPDATE SET
		name = CASE WHEN excluded.name != '' THEN excluded.name ELSE local_chats.name END,
		last_message = CASE WHEN excluded.last_message != '' THEN excluded.last_message ELSE local_chats.last_message END,
		last_message_time = MAX(local_chats.last_message_time, excluded.last_message_time),
		is_group = excluded.is_group
	`
	_, err := s.db.Exec(query, chat.JID, chat.Name, chat.LastMessage, chat.LastMessageTime, chat.IsGroup, chat.UnreadCount)
	return err
}

// UpdateChatLastMessage atualiza a última mensagem e incrementa não lidas se aplicável
func (s *Store) UpdateChatLastMessage(chatJID, name, msg string, t time.Time, isIncoming bool, isCurrentChat bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	unreadIncrement := 0
	if isIncoming && !isCurrentChat {
		unreadIncrement = 1
	}

	query := `
	INSERT INTO local_chats (jid, name, last_message, last_message_time, is_group, unread_count)
	VALUES (?, ?, ?, ?, 0, ?)
	ON CONFLICT(jid) DO UPDATE SET
		name = CASE WHEN excluded.name != '' THEN excluded.name ELSE local_chats.name END,
		last_message = excluded.last_message,
		last_message_time = excluded.last_message_time,
		unread_count = CASE WHEN ? THEN local_chats.unread_count + 1 ELSE local_chats.unread_count END
	`
	_, err := s.db.Exec(query, chatJID, name, msg, t, unreadIncrement, unreadIncrement > 0)
	return err
}

// ResetUnread reseta a contagem de não lidos de uma conversa
func (s *Store) ResetUnread(chatJID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`UPDATE local_chats SET unread_count = 0 WHERE jid = ?`, chatJID)
	return err
}

// UpsertChatsBatch insere ou atualiza uma lista de conversas dentro de uma única transação atômica
func (s *Store) UpsertChatsBatch(chats []*Chat) error {
	if len(chats) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
	INSERT INTO local_chats (jid, name, last_message, last_message_time, is_group, unread_count)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(jid) DO UPDATE SET
		name = CASE WHEN excluded.name != '' THEN excluded.name ELSE local_chats.name END,
		last_message = CASE WHEN excluded.last_message != '' THEN excluded.last_message ELSE local_chats.last_message END,
		last_message_time = MAX(local_chats.last_message_time, excluded.last_message_time),
		is_group = excluded.is_group
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, chat := range chats {
		if _, err := stmt.Exec(chat.JID, chat.Name, chat.LastMessage, chat.LastMessageTime, chat.IsGroup, chat.UnreadCount); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveMessage armazena uma mensagem no banco de dados
func (s *Store) SaveMessage(msg *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	INSERT OR REPLACE INTO local_messages (id, chat_jid, sender_jid, sender_name, text, timestamp, is_from_me, status)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query, msg.ID, msg.ChatJID, msg.SenderJID, msg.SenderName, msg.Text, msg.Timestamp, msg.IsFromMe, msg.Status)
	return err
}

// SaveMessagesBatch armazena múltiplas mensagens em lote em uma única transação
func (s *Store) SaveMessagesBatch(msgs []*Message) error {
	if len(msgs) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
	INSERT OR REPLACE INTO local_messages (id, chat_jid, sender_jid, sender_name, text, timestamp, is_from_me, status)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, msg := range msgs {
		if _, err := stmt.Exec(msg.ID, msg.ChatJID, msg.SenderJID, msg.SenderName, msg.Text, msg.Timestamp, msg.IsFromMe, msg.Status); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetMessages recupera as últimas N mensagens de uma conversa em ordem cronológica crescente
func (s *Store) GetMessages(chatJID string, limit int) ([]*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
	SELECT id, chat_jid, sender_jid, COALESCE(sender_name, ''), text, timestamp, is_from_me, status
	FROM (
		SELECT id, chat_jid, sender_jid, sender_name, text, timestamp, is_from_me, status
		FROM local_messages
		WHERE chat_jid = ?
		ORDER BY timestamp DESC
		LIMIT ?
	) sub
	ORDER BY timestamp ASC
	`
	rows, err := s.db.Query(query, chatJID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ChatJID, &m.SenderJID, &m.SenderName, &m.Text, &m.Timestamp, &m.IsFromMe, &m.Status); err != nil {
			return nil, err
		}
		messages = append(messages, &m)
	}
	return messages, nil
}

// UpdateChatName atualiza o nome de exibição de um contato/chat
func (s *Store) UpdateChatName(chatJID string, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`UPDATE local_chats SET name = ? WHERE jid = ?`, name, chatJID)
	return err
}

// GetChat recupera os dados de uma conversa específica pelo JID
func (s *Store) GetChat(chatJID string) (*Chat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var c Chat
	query := `SELECT jid, name, COALESCE(last_message, ''), last_message_time, is_group, unread_count FROM local_chats WHERE jid = ?`
	err := s.db.QueryRow(query, chatJID).Scan(&c.JID, &c.Name, &c.LastMessage, &c.LastMessageTime, &c.IsGroup, &c.UnreadCount)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Close fecha a conexão com o banco SQLite
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}
