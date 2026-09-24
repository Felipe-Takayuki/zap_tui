package config

import (
	"flag"
	"os"
	"path/filepath"
)

// Config armazena os parâmetros de inicialização do aplicativo
type Config struct {
	DBPath      string
	LogFilePath string
	Debug       bool
}

// LoadConfig carrega as configurações via flags de linha de comando ou padrões
func LoadConfig() *Config {
	homeDir, err := os.UserHomeDir()
	defaultDB := "zap_session.db"
	if err == nil {
		appDir := filepath.Join(homeDir, ".zap_tui")
		_ = os.MkdirAll(appDir, 0750)
		defaultDB = filepath.Join(appDir, "session.db")
	}

	dbPath := flag.String("db", defaultDB, "Caminho do arquivo de banco de dados SQLite para sessão e mensagens")
	logFile := flag.String("log", "", "Caminho do arquivo de log (deixe vazio para desativar logs de rede)")
	debug := flag.Bool("debug", false, "Habilitar modo de depuração")

	flag.Parse()

	return &Config{
		DBPath:      *dbPath,
		LogFilePath: *logFile,
		Debug:       *debug,
	}
}
