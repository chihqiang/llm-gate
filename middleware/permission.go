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
//   - /* 通配模式：去掉 /* 后做路径段前缀匹配
//
// 示例：
//
//	/api/v1/sys/accounts      → 仅匹配 /api/v1/sys/accounts（精确）
//	/api/v1/sys/accounts/*    → 匹配 /api/v1/sys/accounts/1, /api/v1/sys/accounts/1/edit（段前缀）
//	/api/v1/sys/accounts/*    → 不匹配 /api/v1/sys/accounts（需要子路径）
//	/api/v1/sys/accounts      → 不匹配 /api/v1/sys/accounts-extra（段边界保护）
func urlMatch(pattern, uri string) bool {
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(uri, prefix) &&
			len(uri) > len(prefix) &&
			uri[len(prefix)] == '/'
	}
	return uri == pattern
}
