package components

import (
	"zap_tui/internal/ui/styles"
)

// RenderFooter renderiza o rodapé com os atalhos de navegação
func RenderFooter(width int) string {
	renderKey := func(key, desc string) string {
		return styles.ShortcutKeyStyle.Render(key) + " " + desc
	}

	shortcuts := renderKey("[Tab]", "Foco") + "  •  " +
		renderKey("[↑/↓]", "Navegar") + "  •  " +
		renderKey("[Enter]", "Enviar/Abrir") + "  •  " +
		renderKey("[Esc]", "Lista") + "  •  " +
		renderKey("[/to <num> <msg>]", "Novo Chat") + "  •  " +
		renderKey("[Ctrl+C]", "Sair")

	return styles.FooterStyle.Width(width - 2).Render(shortcuts)
}
