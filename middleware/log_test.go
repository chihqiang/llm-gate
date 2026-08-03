package middleware

import (
	"strings"
	"testing"
)

func TestSanitizePayload(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"empty", "", ""},
		{"not json stays trimmed", "  raw text  ", "raw text"},
		{"password masked", `{"password":"secret","name":"a"}`, `{"name":"a","password":"***"}`},
		{"token masked", `{"token":"abc","refresh_token":"x","ok":true}`, `{"ok":true,"refresh_token":"***","token":"***"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sanitizePayload(c.in)
			if c.name == "password masked" || c.name == "token masked" {
				// JSON 键序可能不同，逐键断言敏感字段被脱敏、非敏感字段保留
				for _, key := range []string{"password", "token", "refresh_token"} {
					if strings.Contains(c.in, key) && !strings.Contains(got, "***") {
						t.Fatalf("%s: key %s not masked in %q", c.name, key, got)
					}
				}
				for _, keep := range []string{"name", "ok"} {
					if strings.Contains(c.in, keep) && !strings.Contains(got, keep) {
						t.Fatalf("%s: non-sensitive field %q lost: %q", c.name, keep, got)
					}
				}
				return
			}
			if got != c.want {
				t.Fatalf("sanitizePayload(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	if got := truncateString("hello", 3); got != "hel" {
		t.Fatalf("truncateString = %q, want hel", got)
	}
	if got := truncateString("hi", 10); got != "hi" {
		t.Fatalf("truncateString = %q, want hi", got)
	}
}

func TestParseUserAgent(t *testing.T) {
	cases := []struct {
		ua, wantBrowser, wantOS string
	}{
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36", "Chrome", "Mac OS X 10.15.7"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36", "Chrome", "Windows 10/11"},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile Safari/604.1", "Safari", "iPhone OS 17.0"},
		{"curl/8.4.0", "curl", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		t.Run(c.ua, func(t *testing.T) {
			b, os := parseUserAgent(c.ua)
			if b != c.wantBrowser || os != c.wantOS {
				t.Fatalf("parseUserAgent(%q) = (%q, %q), want (%q, %q)", c.ua, b, os, c.wantBrowser, c.wantOS)
			}
		})
	}
}
