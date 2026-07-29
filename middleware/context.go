package middleware

import (
	"context"

	"chihqiang/llm-gate/model"
)

type contextKey string

const accountContextKey contextKey = "account"

func ContextWithAccount(ctx context.Context, account *model.Account) context.Context {
	return context.WithValue(ctx, accountContextKey, account)
}

func AccountFromContext(ctx context.Context) *model.Account {
	account, _ := ctx.Value(accountContextKey).(*model.Account)
	return account
}
