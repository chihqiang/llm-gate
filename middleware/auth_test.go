package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chihqiang/ten-go/rest/token"

	"chihqiang/llm-gate/config"
	"chihqiang/llm-gate/logic"
)

func TestAuthMiddleware(t *testing.T) {
	tokens, err := logic.NewTokenManager(config.JWTConfig{Secret: "test-secret-123"})
	if err != nil {
		t.Fatalf("new token manager: %v", err)
	}
	pair, err := tokens.GenerateTokenPair(logic.Claims{
		logic.ClaimKeyUserID:   int64(42),
		logic.ClaimKeyUsername: "tester",
	})
	if err != nil {
		t.Fatalf("generate pair: %v", err)
	}

	handler := Auth(tokens)(func(w http.ResponseWriter, r *http.Request) {
		if id, ok := token.Claim(r.Context(), logic.ClaimKeyUserID); !ok || id.(float64) != 42 {
			t.Errorf("user_id claim = %v, %v", id, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"access token", "Bearer " + pair.AccessToken, http.StatusNoContent},
		{"refresh token", "Bearer " + pair.RefreshToken, http.StatusUnauthorized},
		{"missing token", "", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			handler(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}
