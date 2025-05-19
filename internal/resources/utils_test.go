package resources

import (
	"testing"
)

func TestEscapeString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "simple password",
			input:   "password123",
			want:    `password123`,
			wantErr: false,
		},
		{
			name:    "password with quotes",
			input:   `pass"word`,
			want:    `pass\"word`,
			wantErr: false,
		},
		{
			name:    "password with backslash",
			input:   `pass\word`,
			want:    `pass\\word`,
			wantErr: false,
		},
		{
			name: "password with newline",
			input: `pass
word`,
			want:    `pass\nword`,
			wantErr: false,
		},
		{
			name:    "password with special chars",
			input:   `P@ssw0rd!@#$%^&*()`,
			want:    `P@ssw0rd!@#$%^\u0026*()`,
			wantErr: false,
		},
		{
			name:    "complex password",
			input:   `P@ssw0rd!"#$%&'()*+,-./:;<=>?@[\]^_{|}~`,
			want:    "P@ssw0rd!\\\"#$%\\u0026'()*+,-./:;\\u003c=\\u003e?@[\\\\]^_{|}~",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EscapeString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("EscapeString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EscapeString() = %v, want %v", got, tt.want)
				t.Logf("Got bytes: %v", []byte(got))
				t.Logf("Want bytes: %v", []byte(tt.want))
			}
		})
	}
}
