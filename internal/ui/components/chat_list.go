package components

import (
	"fmt"
	"strings"
	"time"

	"zap_tui/internal/db"
	"zap_tui/internal/ui/styles"

	"github.com/charmbracelet/lipgloss"
)

// RenderChatList renderiza a coluna lateral esquerda com as conversas
func RenderChatList(chats []*db.Chat, selectedIdx int, width, height int, focused bool) string {
	innerW := width - 2
	innerH := height - 2
	if innerW < 10 {
		innerW = 10
	}
	if innerH < 4 {
		innerH = 4
	}

	header := styles.ChatListHeaderStyle.Render("CONVERSAS")

	if len(chats) == 0 {
		emptyMsg := lipgloss.NewStyle().
			Foreground(styles.ColorTextMuted).
			Padding(1, 1).
			Render("Nenhuma conversa ainda.\n\nUse:\n/to <numero> <msg>\npara iniciar um chat.")
		
		content := lipgloss.JoinVertical(lipgloss.Left, header, emptyMsg)
		style := styles.ChatListPanelStyle
		if focused {
			style = styles.ChatListPanelFocusStyle
		}
		return style.Width(innerW).Height(innerH).Render(content)
	}

	// Cada item de chat consome 2 linhas na interface
	itemsPerPage := (innerH - 2) / 2
	if itemsPerPage < 1 {
		itemsPerPage = 1
	}

	// Cálculo da janela de rolagem para manter o item selecionado visível
	startIdx := 0
	if selectedIdx >= itemsPerPage {
		startIdx = selectedIdx - itemsPerPage + 1
	}
	endIdx := startIdx + itemsPerPage
	if endIdx > len(chats) {
		endIdx = len(chats)
	}

	var chatLines []string
	for i := startIdx; i < endIdx; i++ {
		chat := chats[i]
		isSelected := (i == selectedIdx)

		name := chat.Name
		if name == "" {
			name = chat.JID
		}

		timeStr := ""
		if !chat.LastMessageTime.IsZero() {
			if chat.LastMessageTime.Day() == time.Now().Day() {
				timeStr = chat.LastMessageTime.Format("15:04")
			} else {
				timeStr = chat.LastMessageTime.Format("02/01")
			}
		}

		// Linha 1: Nome + Horário
		maxNameLen := innerW - lipgloss.Width(timeStr) - 4
		if maxNameLen < 5 {
			maxNameLen = 5
		}
		if len(name) > maxNameLen {
			name = name[:maxNameLen-1] + "…"
		}
		
		line1Space := innerW - lipgloss.Width(name) - lipgloss.Width(timeStr) - 3
		if line1Space < 1 {
			line1Space = 1
		}
		row1 := styles.ChatNameStyle.Render(name) + strings.Repeat(" ", line1Space) + styles.ChatTimeStyle.Render(timeStr)

		// Linha 2: Prévia da última mensagem + Contador de não lidos
		preview := chat.LastMessage
		if preview == "" {
			preview = "(sem mensagens)"
		}
		badge := ""
		if chat.UnreadCount > 0 {
			badge = styles.BadgeStyle.Render(fmt.Sprintf("%d", chat.UnreadCount))
		}

		maxPrevLen := innerW - lipgloss.Width(badge) - 4
		if maxPrevLen < 5 {
			maxPrevLen = 5
		}
		if len(preview) > maxPrevLen {
			preview = preview[:maxPrevLen-1] + "…"
		}

		line2Space := innerW - lipgloss.Width(preview) - lipgloss.Width(badge) - 3
		if line2Space < 1 {
			line2Space = 1
		}
		row2 := styles.ChatPreviewStyle.Render(preview) + strings.Repeat(" ", line2Space) + badge

		itemBlock := lipgloss.JoinVertical(lipgloss.Left, row1, row2)
		if isSelected {
			itemBlock = styles.ChatItemSelectedStyle.Width(innerW - 1).Render(itemBlock)
		} else {
			itemBlock = styles.ChatItemStyle.Width(innerW - 1).Render(itemBlock)
		}

		chatLines = append(chatLines, itemBlock)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, chatLines...)
	content := lipgloss.JoinVertical(lipgloss.Left, header, body)

	style := styles.ChatListPanelStyle
	if focused {
		style = styles.ChatListPanelFocusStyle
	}
	return style.Width(innerW).Height(innerH).Render(content)
}
