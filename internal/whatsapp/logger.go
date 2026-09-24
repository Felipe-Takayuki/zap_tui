package whatsapp

import (
	"fmt"
	"os"
	"sync"
	"time"

	"go.mau.fi/whatsmeow/util/log"
)

// FileLogger implementa waLog.Logger escrevendo logs de rede e protocolo em um arquivo
type FileLogger struct {
	file   *os.File
	module string
	mu     *sync.Mutex
}

// NewLogger cria um logger baseado em arquivo ou no-op
func NewLogger(filePath string, module string) (waLog.Logger, func(), error) {
	if filePath == "" {
		return waLog.Noop, func() {}, nil
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		_ = f.Close()
	}

	logger := &FileLogger{
		file:   f,
		module: module,
		mu:     &sync.Mutex{},
	}
	return logger, cleanup, nil
}

func (l *FileLogger) log(level, msg string, args ...any) {
	if l.file == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	formatted := fmt.Sprintf(msg, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	line := fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, level, l.module, formatted)
	_, _ = l.file.WriteString(line)
}

func (l *FileLogger) Warnf(msg string, args ...any) {
	l.log("WARN", msg, args...)
}

func (l *FileLogger) Errorf(msg string, args ...any) {
	l.log("ERROR", msg, args...)
}

func (l *FileLogger) Infof(msg string, args ...any) {
	l.log("INFO", msg, args...)
}

func (l *FileLogger) Debugf(msg string, args ...any) {
	l.log("DEBUG", msg, args...)
}

func (l *FileLogger) Sub(module string) waLog.Logger {
	return &FileLogger{
		file:   l.file,
		module: fmt.Sprintf("%s/%s", l.module, module),
		mu:     l.mu,
	}
}
