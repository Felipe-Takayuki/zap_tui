package components

import (
	"strings"

	"zap_tui/internal/db"
	"zap_tui/internal/ui/styles"

	"github.com/charmbracelet/lipgloss"
)

// RenderMessageHistory renderiza a lista de mensagens formatada para a Viewport com alta performance
func RenderMessageHistory(messages []*db.Message, width int) string {
	if len(messages) == 0 {
		return lipgloss.NewStyle().
			Foreground(styles.ColorTextMuted).
			Padding(2, 2).
			Render("Nenhuma mensagem nesta conversa ainda.\nDigite algo abaixo para começar a conversar!")
	}

	innerW := width - 2
	if innerW < 20 {
		innerW = 20
	}

	// Largura máxima do balão (70% da viewport ou até 55 caracteres)
	maxBubbleWidth := innerW * 7 / 10
	if maxBubbleWidth < 20 {
		maxBubbleWidth = innerW - 2
	}
	if maxBubbleWidth > 58 {
		maxBubbleWidth = 58
	}

	// Estilo reutilizado para o corpo da mensagem
	contentStyle := lipgloss.NewStyle().
		Width(maxBubbleWidth - 2).
		Foreground(styles.ColorTextPrimary)

	var sb strings.Builder
	sb.Grow(len(messages) * 128)

	for _, msg := range messages {
		timeStr := msg.Timestamp.Format("15:04")

		statusMark := ""
		if msg.IsFromMe {
			switch msg.Status {
			case "DELIVERED", "READ":
				statusMark = " ✓✓"
			case "SENT":
				statusMark = " ✓"
			default:
				statusMark = " ◌"
			}
		}

		senderLabel := msg.SenderName
		if senderLabel == "" {
			if msg.IsFromMe {
				senderLabel = "Você"
			} else {
				senderLabel = msg.SenderJID
			}
		}

		meta := styles.MessageMetaStyle.Render(senderLabel + " • " + timeStr + statusMark)
		body := contentStyle.Render(msg.Text)
		blockContent := lipgloss.JoinVertical(lipgloss.Left, meta, body)

		if msg.IsFromMe {
			bubble := styles.SentMessageBubble.Width(maxBubbleWidth).Render(blockContent)
			bWidth := lipgloss.Width(bubble)
			pad := innerW - bWidth
			if pad > 0 {
				prefix := strings.Repeat(" ", pad)
				lines := strings.Split(bubble, "\n")
				for _, line := range lines {
					sb.WriteString(prefix)
					sb.WriteString(line)
					sb.WriteByte('\n')
				}
			} else {
				sb.WriteString(bubble)
				sb.WriteByte('\n')
			}
		} else {
			bubble := styles.RecvMessageBubble.Width(maxBubbleWidth).Render(blockContent)
			sb.WriteString(bubble)
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}
