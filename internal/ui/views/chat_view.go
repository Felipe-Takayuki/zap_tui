package views

import (
	"zap_tui/internal/db"
	"zap_tui/internal/ui/components"
	"zap_tui/internal/ui/styles"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// FocusArea representa a área que detém o foco de entrada do usuário
type FocusArea int

const (
	FocusList FocusArea = iota
	FocusMessages
	FocusInput
)

// RenderChatView renderiza a tela principal com layout dividido (chats à esquerda, histórico e input à direita)
func RenderChatView(
	width, height int,
	chats []*db.Chat,
	selectedChatIdx int,
	activeChat *db.Chat,
	vp viewport.Model,
	input textinput.Model,
	focus FocusArea,
	status string,
	userName string,
	userJID string,
) string {
	if width < 40 || height < 10 {
		return "Terminal muito pequeno para exibir a interface do WhatsApp. Por favor, redimensione a janela."
	}

	header := components.RenderHeader(width, status, userName, userJID)
	footer := components.RenderFooter(width)

	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	bodyHeight := height - headerHeight - footerHeight
	if bodyHeight < 6 {
		bodyHeight = 6
	}

	// Largura da barra lateral esquerda (entre 26 e 40 caracteres, aprox. 28% da tela)
	leftWidth := width * 28 / 100
	if leftWidth < 26 {
		leftWidth = 26
	}
	if leftWidth > 40 {
		leftWidth = 40
	}
	rightWidth := width - leftWidth

	// 1. Barra lateral de chats
	leftView := components.RenderChatList(chats, selectedChatIdx, leftWidth, bodyHeight, focus == FocusList)

	// 2. Coluna direita (Chat Ativo)
	activeName := "Nenhuma conversa selecionada"
	activeJID := ""
	if activeChat != nil {
		activeName = activeChat.Name
		if activeName == "" {
			activeName = activeChat.JID
		}
		activeJID = activeChat.JID
	}

	rightHeaderContent := "💬 " + activeName
	if activeJID != "" && activeJID != activeName {
		rightHeaderContent += " (" + activeJID + ")"
	}
	rightHeader := styles.ChatViewHeaderStyle.Width(rightWidth - 4).Render(rightHeaderContent)

	// Viewport com as mensagens
	vpView := vp.View()

	// Container do input de texto
	inputStyle := styles.InputContainerStyle
	if focus == FocusInput {
		inputStyle = styles.InputContainerFocusStyle
	}
	inputBox := inputStyle.Width(rightWidth - 4).Render(input.View())

	// Painel direito montado
	rightBodyContent := lipgloss.JoinVertical(lipgloss.Left, rightHeader, vpView, inputBox)
	rightPanelStyle := styles.ChatViewPanelStyle
	if focus == FocusMessages {
		rightPanelStyle = styles.ChatViewPanelFocusStyle
	}
	rightView := rightPanelStyle.Width(rightWidth - 2).Height(bodyHeight - 2).Render(rightBodyContent)

	// Junção de colunas horizontalmente
	body := lipgloss.JoinHorizontal(lipgloss.Top, leftView, rightView)

	// Junção final verticalmente
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
