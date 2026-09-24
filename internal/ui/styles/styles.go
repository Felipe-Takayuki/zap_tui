package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Paleta de Cores WhatsApp Dark Theme
	ColorWhatsAppGreen = lipgloss.Color("#25D366")
	ColorTeal          = lipgloss.Color("#128C7E")
	ColorDarkTeal      = lipgloss.Color("#075E54")
	ColorSentBubble    = lipgloss.Color("#005C4B")
	ColorRecvBubble    = lipgloss.Color("#202C33")
	ColorTextPrimary   = lipgloss.Color("#E9EDEF")
	ColorTextMuted     = lipgloss.Color("#8696A0")
	ColorBorder        = lipgloss.Color("#374248")
	ColorBorderFocus   = lipgloss.Color("#25D366")
	ColorHighlight     = lipgloss.Color("#2A3942")
	ColorOnline        = lipgloss.Color("#10B981")
	ColorOffline       = lipgloss.Color("#EF4444")
	ColorConnecting    = lipgloss.Color("#F59E0B")
	ColorBadge         = lipgloss.Color("#25D366")

	// Estilos Globais
	AppTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorTeal).
			Padding(0, 1)

	StatusOnlineStyle = lipgloss.NewStyle().
				Foreground(ColorOnline).
				Bold(true)

	StatusOfflineStyle = lipgloss.NewStyle().
				Foreground(ColorOffline).
				Bold(true)

	StatusConnectingStyle = lipgloss.NewStyle().
				Foreground(ColorConnecting).
				Bold(true)

	HeaderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(ColorBorder).
			Foreground(ColorTextMuted).
			Padding(0, 1)

	ShortcutKeyStyle = lipgloss.NewStyle().
				Foreground(ColorWhatsAppGreen).
				Bold(true)

	// Estilos do Painel de Conversas (Barra Lateral Esquerda)
	ChatListPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder)

	ChatListPanelFocusStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorderFocus)

	ChatListHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorWhatsAppGreen).
				Padding(0, 1)

	ChatItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	ChatItemSelectedStyle = lipgloss.NewStyle().
				Background(ColorHighlight).
				Bold(true).
				Padding(0, 1)

	ChatNameStyle = lipgloss.NewStyle().
			Foreground(ColorTextPrimary).
			Bold(true)

	ChatTimeStyle = lipgloss.NewStyle().
			Foreground(ColorTextMuted)

	ChatPreviewStyle = lipgloss.NewStyle().
				Foreground(ColorTextMuted)

	BadgeStyle = lipgloss.NewStyle().
			Background(ColorBadge).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 1)

	// Estilos do Painel de Mensagens (Painel Direito)
	ChatViewPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder)

	ChatViewPanelFocusStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorderFocus)

	ChatViewHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorTextPrimary).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(ColorBorder).
				Padding(0, 1)

	// Balões de Mensagens
	SentMessageBubble = lipgloss.NewStyle().
				Background(ColorSentBubble).
				Foreground(ColorTextPrimary).
				Padding(0, 1).
				MarginBottom(1)

	RecvMessageBubble = lipgloss.NewStyle().
				Background(ColorRecvBubble).
				Foreground(ColorTextPrimary).
				Padding(0, 1).
				MarginBottom(1)

	MessageMetaStyle = lipgloss.NewStyle().
				Foreground(ColorTextMuted)

	// Input de Texto
	InputContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder).
				Padding(0, 1)

	InputContainerFocusStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(ColorBorderFocus).
					Padding(0, 1)

	// QR Code Screen Styles
	QRBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorWhatsAppGreen).
			Padding(1, 2).
			Align(lipgloss.Center)

	QRTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhatsAppGreen).
			MarginBottom(1)

	QRInstructionStyle = lipgloss.NewStyle().
				Foreground(ColorTextPrimary).
				MarginBottom(1)
)
