package route

import (
	"chihqiang/llm-gate/handler"
	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/jwt"
)

func Register(server *httpx.Server, j *jwt.JWT,
	authSvc *logic.AuthLogic,
	logLogic *logic.LogLogic,
	authHandler *handler.AuthHandler,
	accountHandler *handler.AccountHandler,
	roleHandler *handler.RoleHandler,
	menuHandler *handler.MenuHandler,
	logHandler *handler.LogHandler,
) {
	// 全局中间件
	server.Use(httpx.WithCors("*"))
	server.Use(httpx.WithRecovery())
	server.Use(httpx.WithLogger())
	server.Use(middleware.Log(logLogic, []string{"/health", "/api/v1/sys/logs"}, []string{"OPTIONS", "HEAD"}))

	authMw := middleware.Auth(j)
	loadAccountMw := middleware.LoadAccount(authSvc)

	v1 := server.Group("/api/v1")

	// 公开路由
	v1.AddRoute(httpx.Route{Method: "POST", Path: "/auth/login", Handler: authHandler.Login})

	// 需要鉴权的路由
	permMw := middleware.Permission("/api/v1/auth/me")
	auth := v1.Group("", authMw, loadAccountMw, permMw)
	auth.AddRoutes([]httpx.Route{
		{Method: "GET", Path: "/auth/me", Handler: authHandler.Me},
	})

	auth.AddRoutes([]httpx.Route{
		{Method: "GET", Path: "/sys/accounts", Handler: accountHandler.List},
		{Method: "GET", Path: "/sys/accounts/{id}", Handler: accountHandler.Detail},
		{Method: "POST", Path: "/sys/accounts", Handler: accountHandler.Create},
		{Method: "PUT", Path: "/sys/accounts/{id}", Handler: accountHandler.Update},
		{Method: "DELETE", Path: "/sys/accounts/{id}", Handler: accountHandler.Delete},
	})

	auth.AddRoutes([]httpx.Route{
		{Method: "GET", Path: "/sys/roles", Handler: roleHandler.List},
		{Method: "GET", Path: "/sys/roles/all", Handler: roleHandler.AllList},
		{Method: "GET", Path: "/sys/roles/{id}", Handler: roleHandler.Detail},
		{Method: "POST", Path: "/sys/roles", Handler: roleHandler.Create},
		{Method: "PUT", Path: "/sys/roles/{id}", Handler: roleHandler.Update},
		{Method: "DELETE", Path: "/sys/roles/{id}", Handler: roleHandler.Delete},
		{Method: "POST", Path: "/sys/roles/{id}/menus", Handler: roleHandler.AssociateMenus},
	})

	auth.AddRoutes([]httpx.Route{
		{Method: "GET", Path: "/sys/menus", Handler: menuHandler.List},
		{Method: "GET", Path: "/sys/menus/all", Handler: menuHandler.AllList},
		{Method: "GET", Path: "/sys/menus/{id}", Handler: menuHandler.Detail},
		{Method: "POST", Path: "/sys/menus", Handler: menuHandler.Create},
		{Method: "PUT", Path: "/sys/menus/{id}", Handler: menuHandler.Update},
		{Method: "DELETE", Path: "/sys/menus/{id}", Handler: menuHandler.Delete},
	})

	auth.AddRoutes([]httpx.Route{
		{Method: "GET", Path: "/sys/logs", Handler: logHandler.List},
	})
}
