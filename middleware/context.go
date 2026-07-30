package middleware

import (
	"context"
	"strings"

	"chihqiang/llm-gate/model"
)

type contextKey string

const (
	accountContextKey contextKey = "account"
	adminRoleIDKey    contextKey = "admin_role_id"
	permissionSetKey  contextKey = "permission_set"
)

func ContextWithAccount(ctx context.Context, account *model.Account, adminRoleID int64) context.Context {
	ctx = context.WithValue(ctx, accountContextKey, account)
	ctx = context.WithValue(ctx, adminRoleIDKey, adminRoleID)
	return ctx
}

func AccountFromContext(ctx context.Context) *model.Account {
	account, _ := ctx.Value(accountContextKey).(*model.Account)
	return account
}

func IsAdmin(ctx context.Context) bool {
	account := AccountFromContext(ctx)
	if account == nil {
		return false
	}
	adminRoleID, _ := ctx.Value(adminRoleIDKey).(int64)
	if adminRoleID == 0 {
		return false
	}
	for _, role := range account.Roles {
		if role.ID == adminRoleID {
			return true
		}
	}
	return false
}

// PermissionEntry 表示一条权限规则（方法 + URL 模式）
type PermissionEntry struct {
	Method string
	URL    string
}

// PermissionSet 是预构建的去重权限集合，避免每次请求重复遍历和去重
type PermissionSet struct {
	entries []PermissionEntry
}

// NewPermissionSet 从账号的角色和菜单中预构建去重的权限集合
func NewPermissionSet(account *model.Account) *PermissionSet {
	seen := make(map[string]bool)
	var entries []PermissionEntry
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
			entries = append(entries, PermissionEntry{Method: menu.APIMethod, URL: menu.APIURL})
		}
	}
	return &PermissionSet{entries: entries}
}

// Check 检查请求方法+URL 是否匹配任意一条权限规则
func (ps *PermissionSet) Check(method, uri string) bool {
	for _, e := range ps.entries {
		if methodMatch(e.Method, method) && urlMatch(e.URL, uri) {
			return true
		}
	}
	return false
}

func ContextWithPermissionSet(ctx context.Context, ps *PermissionSet) context.Context {
	return context.WithValue(ctx, permissionSetKey, ps)
}

func PermissionSetFromContext(ctx context.Context) *PermissionSet {
	ps, _ := ctx.Value(permissionSetKey).(*PermissionSet)
	return ps
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
