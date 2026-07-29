package middleware

import (
	"net/http"
	"strings"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/jwt"
)

func Auth(j *jwt.JWT) httpx.Middleware {
	getToken := func(r *http.Request) string {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			return ""
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return ""
		}
		return parts[1]
	}
	return j.AuthMiddleware(getToken)
}
