package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"zap_tui/internal/config"
	"zap_tui/internal/db"
	"zap_tui/internal/ui"
	"zap_tui/internal/whatsapp"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg := config.LoadConfig()

	// 1. Inicializa o banco de dados SQLite local
	store, err := db.NewStore(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao inicializar banco local SQLite: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// 2. Variável ponte para envio thread-safe de eventos para o Bubble Tea
	var p *tea.Program
	dispatcher := func(msg any) {
		if p != nil {
			p.Send(msg)
		}
	}

	// 3. Inicializa o cliente whatsmeow com o banco de dados e o despachador
	client, err := whatsapp.NewClient(cfg.DBPath, cfg.LogFilePath, store, dispatcher)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao inicializar cliente WhatsApp: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// 4. Captura sinais do sistema operacional para encerramento gracioso
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		if p != nil {
			p.Quit()
		}
	}()

	// 5. Inicializa o modelo de interface do Bubble Tea
	model := ui.NewModel(client, store)

	// 6. Configura o programa Bubble Tea com modo de tela cheia (AltScreen)
	p = tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// 7. Inicia a conexão do WhatsApp em background
	go func() {
		if err := client.Connect(context.Background()); err != nil {
			dispatcher(whatsapp.MsgError{Err: err})
		}
	}()

	// 8. Executa a aplicação TUI no terminal
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erro na execução da interface de terminal: %v\n", err)
		os.Exit(1)
	}
}
