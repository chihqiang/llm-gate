package middleware

import (
	"net/http"

	"chihqiang/llm-gate/logic"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/jwt"
	"github.com/chihqiang/infra-go/logger"
)

func LoadAccount(authSvc *logic.AuthLogic, adminRoleID int64) httpx.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			claims := jwt.ClaimsFromContext(r.Context())
			if claims == nil {
				next(w, r)
				return
			}

			id, ok := claims[jwt.ClaimKeyUserID].(float64)
			if !ok || id == 0 {
				next(w, r)
				return
			}

			account, err := authSvc.GetAccountByID(int64(id))
			if err != nil {
				logger.Error("load account failed", logger.Err(err), logger.Int64("account_id", int64(id)))
				httpx.WriteHTTPError(w, httpx.CodeDefaultError, "获取用户信息失败")
				return
			}

			ctx := ContextWithAccount(r.Context(), account, adminRoleID)
			// 预构建权限集合，避免在 Permission 中间件中每次重新遍历和去重
			ctx = ContextWithPermissionSet(ctx, NewPermissionSet(account))
			next(w, r.WithContext(ctx))
		}
	}
}
