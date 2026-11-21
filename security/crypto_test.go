package security

import (
	"testing"
)

func TestCipherRoundTrip(t *testing.T) {
	c, err := New("", "test-jwt-secret-key-for-encryption")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := []struct {
		name, plain string
	}{
		{"empty", ""},
		{"short", "sk-abc"},
		{"long unicode", "密钥&%#@中文符号: abc123"},
		{"apikey like", "sk-proj-abcdefghijklmnopqrstuvwxyz0123456789"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			enc, err := c.Encrypt(tc.plain)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			if tc.plain == "" && enc != "" {
				t.Fatalf("empty plaintext should encrypt to empty, got %q", enc)
			}
			dec, err := c.Decrypt(enc)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if dec != tc.plain {
				t.Fatalf("round trip mismatch: got %q, want %q", dec, tc.plain)
			}
		})
	}
}

func TestCipherDecryptPlaintextFallback(t *testing.T) {
	c, err := New("", "key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// 无前缀的字符串应按原值返回（兼容历史明文）
	got, err := c.Decrypt("sk-plain-text")
	if err != nil || got != "sk-plain-text" {
		t.Fatalf("Decrypt legacy plaintext = %q, %v; want sk-plain-text, nil", got, err)
	}
	if IsEncrypted("sk-plain-text") {
		t.Fatalf("IsEncrypted should be false for plaintext")
	}
}

func TestCipherTampered(t *testing.T) {
	c, err := New("", "key")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	enc, err := c.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	// 篡改密文最后一个字符，解密必须失败
	b := []byte(enc)
	if b[len(b)-1] == 'A' {
		b[len(b)-1] = 'B'
	} else {
		b[len(b)-1] = 'A'
	}
	if _, err := c.Decrypt(string(b)); err == nil {
		t.Fatalf("Decrypt of tampered ciphertext should fail")
	}
}
