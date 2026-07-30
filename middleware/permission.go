package middleware

import (
	"net/http"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/logger"
)

func Permission(skipRoutes ...string) httpx.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			uri := r.URL.Path
			for _, route := range skipRoutes {
				if uri == route {
					next(w, r)
					return
				}
			}

			account := AccountFromContext(r.Context())
			if account == nil {
				httpx.WriteHTTPError(w, httpx.CodeUnauthorized, "未登录")
				return
			}

			if !account.Status {
				httpx.WriteHTTPError(w, httpx.CodeUnauthorized, "账号已被禁用")
				return
			}

			ps := PermissionSetFromContext(r.Context())
			if ps == nil || !ps.Check(r.Method, uri) {
				logger.Warn("permission denied",
					logger.Int64("account_id", account.ID),
					logger.String("method", r.Method),
					logger.String("uri", uri),
				)
				httpx.WriteHTTPError(w, httpx.CodeForbidden, "无权限访问")
				return
			}

			next(w, r)
		}
	}
}
