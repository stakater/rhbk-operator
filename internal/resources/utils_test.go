package resources

import (
	"bytes"
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
			name:    "password with windows line ending",
			input:   "pass\r\nword",
			want:    `pass\r\nword`,
			wantErr: false,
		},
		{
			name:    "complex password",
			input:   `P@ssw0rd!"#$%&'()*+,-./:;<=>?@[\]^_{|}~`,
			want:    "P@ssw0rd!\\\"#$%\\u0026'()*+,-./:;\\u003c=\\u003e?@[\\\\]^_{|}~",
			wantErr: false,
		},
		{
			name:    "pem certificate format",
			input:   "-----BEGIN CERTIFICATE-----\nMIIFazCCA1OgAwIBAgIRAIIQz7DSQONZRGPgu2OCiwAwDQYJKoZIhvcNAQELBQAw\nTzELMAkGA1UEBhMCVVMxKTAnBgNVBAoTIEludGVybmV0IFNlY3VyaXR5IFJlc2Vh\ncmNoIEdyb3VwMRUwEwYDVQQDEwxJU1JHIFJvb3QgWDEwHhcNMTUwNjA0MTEwNDM4\nWhcNMzUwNjA0MTEwNDM4WjBPMQswCQYDVQQGEwJVUzEpMCcGA1UEChMgSW50ZXJu\nZXQgU2VjdXJpdHkgUmVzZWFyY2ggR3JvdXAxFTATBgNVBAMTDElTUkcgUm9vdCBY\nMTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAK3oJHP0FDfzm54rVygc\nh77ct984kIxuPOZXoHj3dcKi/v1q1HpWi7i56j3j6KR1xjvn7p9aCWcRxPFhXFqM\n47OhiDijXa+sRporq0Wgx//hkuSzWHznYy2h2k7RkZinljCu2XnUlpfMk6Wcti8p\nDePeaa2it5u7GiNmwUjr0t6U14UoX6Hn4OBUBarjQ6/btjcaJiVKiH94jHg9zd2s\n-----END CERTIFICATE-----",
			want:    "-----BEGIN CERTIFICATE-----\nMIIFazCCA1OgAwIBAgIRAIIQz7DSQONZRGPgu2OCiwAwDQYJKoZIhvcNAQELBQAw\nTzELMAkGA1UEBhMCVVMxKTAnBgNVBAoTIEludGVybmV0IFNlY3VyaXR5IFJlc2Vh\ncmNoIEdyb3VwMRUwEwYDVQQDEwxJU1JHIFJvb3QgWDEwHhcNMTUwNjA0MTEwNDM4\nWhcNMzUwNjA0MTEwNDM4WjBPMQswCQYDVQQGEwJVUzEpMCcGA1UEChMgSW50ZXJu\nZXQgU2VjdXJpdHkgUmVzZWFyY2ggR3JvdXAxFTATBgNVBAMTDElTUkcgUm9vdCBY\nMTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAK3oJHP0FDfzm54rVygc\nh77ct984kIxuPOZXoHj3dcKi/v1q1HpWi7i56j3j6KR1xjvn7p9aCWcRxPFhXFqM\n47OhiDijXa+sRporq0Wgx//hkuSzWHznYy2h2k7RkZinljCu2XnUlpfMk6Wcti8p\nDePeaa2it5u7GiNmwUjr0t6U14UoX6Hn4OBUBarjQ6/btjcaJiVKiH94jHg9zd2s\n-----END CERTIFICATE-----",
			wantErr: false,
		},
		{
			name:    "pem certificate format with windows line endings",
			input:   "-----BEGIN CERTIFICATE-----\r\nMIIFazCCA1OgAwIBAgIRAIIQz7DSQONZRGPgu2OCiwAwDQYJKoZIhvcNAQELBQAw\r\nTzELMAkGA1UEBhMCVVMxKTAnBgNVBAoTIEludGVybmV0IFNlY3VyaXR5IFJlc2Vh\r\ncmNoIEdyb3VwMRUwEwYDVQQDEwxJU1JHIFJvb3QgWDEwHhcNMTUwNjA0MTEwNDM4\r\nWhcNMzUwNjA0MTEwNDM4WjBPMQswCQYDVQQGEwJVUzEpMCcGA1UEChMgSW50ZXJu\r\nZXQgU2VjdXJpdHkgUmVzZWFyY2ggR3JvdXAxFTATBgNVBAMTDElTUkcgUm9vdCBY\r\nMTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAK3oJHP0FDfzm54rVygc\r\nh77ct984kIxuPOZXoHj3dcKi/v1q1HpWi7i56j3j6KR1xjvn7p9aCWcRxPFhXFqM\r\n47OhiDijXa+sRporq0Wgx//hkuSzWHznYy2h2k7RkZinljCu2XnUlpfMk6Wcti8p\r\nDePeaa2it5u7GiNmwUjr0t6U14UoX6Hn4OBUBarjQ6/btjcaJiVKiH94jHg9zd2s\r\n-----END CERTIFICATE-----",
			want:    "-----BEGIN CERTIFICATE-----\r\nMIIFazCCA1OgAwIBAgIRAIIQz7DSQONZRGPgu2OCiwAwDQYJKoZIhvcNAQELBQAw\r\nTzELMAkGA1UEBhMCVVMxKTAnBgNVBAoTIEludGVybmV0IFNlY3VyaXR5IFJlc2Vh\r\ncmNoIEdyb3VwMRUwEwYDVQQDEwxJU1JHIFJvb3QgWDEwHhcNMTUwNjA0MTEwNDM4\r\nWhcNMzUwNjA0MTEwNDM4WjBPMQswCQYDVQQGEwJVUzEpMCcGA1UEChMgSW50ZXJu\r\nZXQgU2VjdXJpdHkgUmVzZWFyY2ggR3JvdXAxFTATBgNVBAMTDElTUkcgUm9vdCBY\r\nMTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAK3oJHP0FDfzm54rVygc\r\nh77ct984kIxuPOZXoHj3dcKi/v1q1HpWi7i56j3j6KR1xjvn7p9aCWcRxPFhXFqM\r\n47OhiDijXa+sRporq0Wgx//hkuSzWHznYy2h2k7RkZinljCu2XnUlpfMk6Wcti8p\r\nDePeaa2it5u7GiNmwUjr0t6U14UoX6Hn4OBUBarjQ6/btjcaJiVKiH94jHg9zd2s\r\n-----END CERTIFICATE-----",
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
			gotBytes := []byte(got)
			wantBytes := []byte(tt.want)
			if !bytes.Equal(gotBytes, wantBytes) {
				t.Errorf("EscapeString() bytes = %v, want %v", gotBytes, wantBytes)
				t.Logf("Got string: %q", got)
				t.Logf("Want string: %q", tt.want)
			}
		})
	}
}
