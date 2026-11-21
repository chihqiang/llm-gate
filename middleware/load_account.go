package middleware

import (
	"net/http"

	"chihqiang/llm-gate/logic"

	"github.com/chihqiang/ten-go/core/logger"
	"github.com/chihqiang/ten-go/rest"
	"github.com/chihqiang/ten-go/rest/chain"
	"github.com/chihqiang/ten-go/rest/token"
)

func LoadAccount(authSvc *logic.AuthLogic, adminRoleID int64) chain.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			idVal, found := token.Claim(r.Context(), logic.ClaimKeyUserID)
			if !found {
				next(w, r)
				return
			}
			id, ok := idVal.(float64)
			if !ok || id == 0 {
				next(w, r)
				return
			}

			account, err := authSvc.GetAccountByID(r.Context(), int64(id))
			if err != nil {
				logger.ErrorCtx(r.Context(), "load account failed", logger.Err(err), logger.Int64("account_id", int64(id)))
				rest.ErrorJSONCtx(r.Context(), w, rest.NewCodeError(rest.BusinessCodeError, "获取用户信息失败"))
				return
			}

			ctx := ContextWithAccount(r.Context(), account, adminRoleID)
			// 预构建权限集合，避免在 Permission 中间件中每次重新遍历和去重
			ctx = ContextWithPermissionSet(ctx, NewPermissionSet(account))
			next(w, r.WithContext(ctx))
		}
	}
}
