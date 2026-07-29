package middleware

import (
	"net/http"
	"strings"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/logger"
)

func Permission(skipRoutes ...string) httpx.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			uri := r.RequestURI
			if idx := strings.IndexByte(uri, '?'); idx > 0 {
				uri = uri[:idx]
			}
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

			if len(account.Roles) == 0 {
				httpx.WriteHTTPError(w, httpx.CodeForbidden, "无权限访问")
				return
			}

			method := r.Method

			seen := make(map[string]bool)
			for _, role := range account.Roles {
				for _, menu := range role.Menus {
					if menu.APIMethod == "" || menu.APIURL == "" {
						continue
					}
					key := menu.APIMethod + " " + menu.APIURL
					if seen[key] {
						continue
					}
					seen[key] = true
					if methodMatch(menu.APIMethod, method) && urlMatch(menu.APIURL, uri) {
						next(w, r)
						return
					}
				}
			}

			logger.Warn("permission denied",
				logger.Int64("account_id", account.ID),
				logger.String("method", method),
				logger.String("uri", r.RequestURI),
			)

			httpx.WriteHTTPError(w, httpx.CodeForbidden, "无权限访问")
		}
	}
}

// methodMatch 检查请求方法是否匹配权限定义的方法
// 支持 "*" 通配（匹配所有方法）
func methodMatch(pattern, method string) bool {
	return pattern == "*" || pattern == method
}

// urlMatch 检查请求 URI 是否匹配权限定义的 APIURL
//   - 非通配模式：必须完全相等
//   - /* 后缀通配：匹配任意子路径（/api/v1/accounts/* → /api/v1/accounts/1）
//   - * 段通配：匹配路径中单个段（/api/v1/providers/*/sync-models → /api/v1/providers/1/sync-models）
func urlMatch(pattern, uri string) bool {
	if !strings.Contains(pattern, "*") {
		return uri == pattern
	}

	// /* 后缀通配（原有逻辑）
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(uri, prefix) &&
			len(uri) > len(prefix) &&
			uri[len(prefix)] == '/'
	}

	// 段通配：按 / 分割，逐段比较
	pParts := strings.Split(pattern, "/")
	uParts := strings.Split(uri, "/")
	if len(pParts) != len(uParts) {
		return false
	}
	for i := range pParts {
		if pParts[i] == "*" {
			continue
		}
		if pParts[i] != uParts[i] {
			return false
		}
	}
	return true
}
