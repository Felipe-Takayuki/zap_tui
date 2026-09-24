package whatsapp

import (
	"bytes"

	"github.com/mdp/qrterminal/v3"
)

// GenerateQRAscii converte a string do QR Code do WhatsApp em blocos ANSI compactos para o terminal
func GenerateQRAscii(code string) string {
	var buf bytes.Buffer
	config := qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     &buf,
		HalfBlocks: true,
		QuietZone:  1,
	}
	qrterminal.GenerateWithConfig(code, config)
	return buf.String()
}
