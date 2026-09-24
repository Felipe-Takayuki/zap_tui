# 📱 ZAP TUI - Terminal User Interface para WhatsApp

Aplicação de terminal (TUI) em Go para utilização do WhatsApp via terminal, integrando a biblioteca [`go.mau.fi/whatsmeow`](https://github.com/tulir/whatsmeow) com o framework [`Bubble Tea`](https://github.com/charmbracelet/bubbletea) e estilização moderna via [`Lipgloss`](https://github.com/charmbracelet/lipgloss).

---

## 🛠️ Tecnologias Utilizadas

- **Linguagem**: Go (Go Modules, compatível com 1.21+)
- **WhatsApp Engine**: [`go.mau.fi/whatsmeow`](https://github.com/tulir/whatsmeow)
- **Banco de Dados Local**: SQLite CGO-free via [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite)
- **Framework TUI**: Charm CLI ([`bubbletea`](https://github.com/charmbracelet/bubbletea), [`bubbles`](https://github.com/charmbracelet/bubbles), [`lipgloss`](https://github.com/charmbracelet/lipgloss))
- **Renderização de QR Code**: ANSI Half-blocks via [`github.com/mdp/qrterminal/v3`](https://github.com/mdp/qrterminal)

---

## 📐 Arquitetura do Projeto

O projeto adota uma separação estrita entre a camada de rede/protocolo e a camada de renderização, garantindo que não ocorram *race conditions* de concorrência:

```
┌─────────────────────────────────┐
│     whatsmeow Event Handler     │ (Goroutines internas de rede)
└────────────────┬────────────────┘
                 │
                 │ Thread-safe via tea.Program.Send(Msg)
                 ▼
┌─────────────────────────────────┐
│     Bubble Tea Update Loop      │ (Execução serial/síncrona na goroutine da TUI)
└────────────────┬────────────────┘
                 │
                 ▼
┌─────────────────────────────────┐
│         SQLite Storage          │ (WAL Mode: sessão whatsmeow + mensagens locais)
└─────────────────────────────────┘
```

### Estrutura de Diretórios
```
zap_tui/
├── cmd/
│   └── zap_tui/
│       └── main.go                  # Ponto de entrada e bootstrap da aplicação
├── internal/
│   ├── config/
│   │   └── config.go                # Flags de inicialização e configurações
│   ├── db/
│   │   ├── models.go                # Modelos de dados locais (Chat, Message)
│   │   ├── store.go                 # Gerenciamento SQLite local com WAL
│   │   └── store_test.go            # Testes de persistência
│   ├── whatsapp/
│   │   ├── client.go                # Gerenciador de conexão whatsmeow e envio
│   │   ├── events.go                # Tipos de mensagens desacopladas para a UI
│   │   ├── logger.go                # Logger em arquivo (evita poluir o stdout)
│   │   ├── qr.go                    # Formatador de QR Code em blocos ANSI
│   │   └── client_test.go           # Testes unitários de parsing de JID/telefone
│   └── ui/
│       ├── model.go                 # Modelo central do Bubble Tea (Update/View)
│       ├── styles/
│       │   └── styles.go            # Paleta de cores (Dark Theme WhatsApp) e Lipgloss
│       ├── components/
│       │   ├── header.go            # Barra superior (status de conexão e usuário)
│       │   ├── footer.go            # Rodapé com legenda de atalhos
│       │   ├── chat_list.go         # Coluna lateral com resumo de conversas e badges
│       │   └── message_view.go      # Renderizador de balões de mensagens (enviadas/recebidas)
│       └── views/
│           ├── qr_view.go           # Tela de autenticação por QR Code
│           └── chat_view.go         # Layout principal dividido (split 2 colunas)
├── go.mod
├── go.sum
└── README.md
```

---

## 🚀 Como Executar

### 1. Pré-requisitos
- Go 1.21 ou superior instalado.
- Terminal compatível com caracteres Unicode e cores ANSI (ex: Kitty, Alacritty, WezTerm, iTerm2, GNOME Terminal, Windows Terminal).

### 2. Clonar ou Acessar o Repositório
```bash
cd zap_tui
```

### 3. Baixar Dependências
```bash
go mod tidy
```

### 4. Executar os Testes
```bash
go test ./...
```

### 5. Compilar e Executar
```bash
# Execução direta:
go run ./cmd/zap_tui

# Ou compilar o binário:
go build -o zap_tui ./cmd/zap_tui
./zap_tui
```

#### Flags Disponíveis:
- `-db <caminho>`: Caminho do arquivo SQLite (padrão: `~/.zap_tui/session.db`).
- `-log <caminho>`: Gravar logs de rede do WhatsApp em um arquivo texto (ex: `-log zap.log`).
- `-debug`: Ativa logs detalhados de depuração.

---

## ⌨️ Navegação e Atalhos

| Tecla / Comando | Ação |
|---|---|
| `Tab` | Alterna o foco sequencialmente entre: Lista de Conversas ➔ Campo de Mensagem ➔ Histórico de Mensagens |
| `Shift + Tab` | Alterna o foco na ordem inversa |
| `↑` / `↓` ou `k` / `j` | Navega entre as conversas (quando na lista) ou rola o histórico (quando nas mensagens) |
| `Enter` | Na lista: Abre a conversa selecionada. No campo de texto: Envia a mensagem digitada |
| `Esc` | Retorna o foco diretamente para a lista de conversas |
| `/to <número> <mensagem>` | Envia mensagem diretamente para um novo número (ex: `/to 5511999998888 Olá!`) |
| `/new <número>` | Abre uma nova conversa com o número informado |
| `Ctrl + C` | Encerramento seguro da aplicação (fecha a conexão e salva o estado do banco SQLite) |

---

## 🔒 Armazenamento de Sessão

A sessão é armazenada localmente em um banco de dados SQLite sem necessidade de compilação CGO (`modernc.org/sqlite`).
- Na **primeira execução**, um QR Code será renderizado na tela. Basta abrir o WhatsApp no seu smartphone (`Aparelhos conectados > Conectar um aparelho`) e escanear o terminal.
- Nas **execuções seguintes**, a aplicação reconhece automaticamente a sessão salva e conecta imediatamente sem necessidade de novo escaneamento.
