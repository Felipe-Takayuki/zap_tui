package components

import (
	"strings"

	"zap_tui/internal/ui/styles"

	"github.com/charmbracelet/lipgloss"
)

// RenderHeader renderiza a barra superior de status do aplicativo
func RenderHeader(width int, status string, userName, userJID string) string {
	appTitle := styles.AppTitleStyle.Render("📱 ZAP TUI")

	userInfo := ""
	if userName != "" || userJID != "" {
		display := userName
		if display == "" {
			display = userJID
		} else if userJID != "" {
			display = display + " (" + userJID + ")"
		}
		userInfo = lipgloss.NewStyle().Foreground(styles.ColorTextMuted).Render("👤 " + display)
	}

	var statusBadge string
	switch strings.ToLower(status) {
	case "conectado", "online":
		statusBadge = styles.StatusOnlineStyle.Render("● Conectado")
	case "conectando...", "reconectando...":
		statusBadge = styles.StatusConnectingStyle.Render("◌ " + status)
	default:
		statusBadge = styles.StatusOfflineStyle.Render("○ " + status)
	}

	leftAndCenter := appTitle
	if userInfo != "" {
		leftAndCenter += "  " + userInfo
	}

	leftLen := lipgloss.Width(leftAndCenter)
	rightLen := lipgloss.Width(statusBadge)

	space := width - leftLen - rightLen - 2
	if space < 1 {
		space = 1
	}

	content := leftAndCenter + strings.Repeat(" ", space) + statusBadge
	return styles.HeaderStyle.Width(width - 2).Render(content)
}
