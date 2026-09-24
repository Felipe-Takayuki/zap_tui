package whatsapp

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestParseTargetJID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantUser string
		wantErr  bool
	}{
		{
			name:     "Número com DDI e DDD simples",
			input:    "5511999998888",
			wantUser: "5511999998888",
			wantErr:  false,
		},
		{
			name:     "Número formatado com símbolos",
			input:    "+55 (11) 99999-8888",
			wantUser: "5511999998888",
			wantErr:  false,
		},
		{
			name:     "JID completo de usuário",
			input:    "5511999998888@s.whatsapp.net",
			wantUser: "5511999998888",
			wantErr:  false,
		},
		{
			name:     "JID de grupo",
			input:    "1203630283921@g.us",
			wantUser: "1203630283921",
			wantErr:  false,
		},
		{
			name:    "Número muito curto",
			input:   "1234",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTargetJID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseTargetJID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if got.User != tt.wantUser {
					t.Errorf("got user = %s, want %s", got.User, tt.wantUser)
				}
				if got.Server != types.DefaultUserServer && got.Server != types.GroupServer {
					t.Errorf("unexpected server: %s", got.Server)
				}
			}
		})
	}
}
