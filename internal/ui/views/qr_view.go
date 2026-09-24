package views

import (
	"zap_tui/internal/ui/styles"

	"github.com/charmbracelet/lipgloss"
)

// RenderQRView renderiza a tela de boas-vindas e pareamento de QR Code
func RenderQRView(width, height int, qrAscii string, status string) string {
	title := styles.QRTitleStyle.Render("🔐 AUTENTICAÇÃO WHATSAPP")

	instructions := styles.QRInstructionStyle.Render(
		"1. Abra o WhatsApp no seu telefone\n" +
			"2. Toque em Mais opções (⋮) ou Configurações\n" +
			"3. Selecione Aparelhos conectados > Conectar um aparelho\n" +
			"4. Aponte a câmera para o QR Code abaixo:",
	)

	qrDisplay := qrAscii
	if qrDisplay == "" {
		qrDisplay = lipgloss.NewStyle().
			Foreground(styles.ColorConnecting).
			Padding(2, 4).
			Render("Gerando QR Code... Aguarde alguns instantes.")
	}

	statusText := lipgloss.NewStyle().
		Foreground(styles.ColorTextMuted).
		MarginTop(1).
		Render(status + "\n[Pressione Ctrl+C para cancelar]")

	boxContent := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		instructions,
		qrDisplay,
		statusText,
	)

	box := styles.QRBoxStyle.Render(boxContent)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}
