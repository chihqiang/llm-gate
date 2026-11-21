package middleware

import (
	"errors"
	"net/http"

	jwtlib "github.com/golang-jwt/jwt/v5"

	"github.com/chihqiang/ten-go/rest"
	"github.com/chihqiang/ten-go/rest/chain"
	resthandler "github.com/chihqiang/ten-go/rest/handler"
	"github.com/chihqiang/ten-go/rest/token"

	"chihqiang/llm-gate/logic"
)

// Auth 管理端 JWT 鉴权中间件，直接复用 ten-go rest/handler 的 AuthHandler：
// 校验签名/有效期后，把业务声明写入请求上下文（下游用 token.Claim 读取）。
func Auth(tokens *logic.TokenManager) chain.Middleware {
	return resthandler.NewAuthHandler(resthandler.AuthConfig{
		Secret: tokens.AccessSecret(),
		Callback: func(w http.ResponseWriter, r *http.Request, err error) {
			msg := "invalid token"
			switch {
			case errors.Is(err, resthandler.ErrMalformedAuthorization):
				msg = "malformed authorization header"
			case errors.Is(err, token.ErrMissingToken):
				msg = "token is missing"
			case errors.Is(err, jwtlib.ErrTokenExpired):
				msg = "token expired"
			}
			_ = rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(http.StatusUnauthorized, msg))
		},
	}).Handle
}
