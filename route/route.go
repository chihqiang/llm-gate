package route

import (
	"net/http"

	"chihqiang/llm-gate/config"
	"chihqiang/llm-gate/handler"
	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/middleware"
	"chihqiang/llm-gate/relay"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/jwt"
)

func Register(server *httpx.Server, j *jwt.JWT,
	authSvc *logic.AuthLogic,
	logLogic *logic.LogLogic,
	cfg config.Config,
	authHandler *handler.AuthHandler,
	accountHandler *handler.AccountHandler,
	roleHandler *handler.RoleHandler,
	menuHandler *handler.MenuHandler,
	logHandler *handler.LogHandler,
	dashboardHandler *handler.DashboardHandler,
	billingHandler *handler.BillingHandler,

	providerHandler *handler.ProviderHandler,
	modelHandler *handler.ModelHandler,
	tokenHandler *handler.TokenHandler,
	usageHandler *handler.UsageHandler,
	relayHandler *relay.RelayHandler,
) {
	if cfg.Pprof.Enabled {
		server.AddRoutes(httpx.PprofRoutes(""))
	}

	server.AddRoute(httpx.Route{
		Method: http.MethodGet,
		Path:   "/health",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			httpx.OkJSON(w, map[string]string{"status": "ok"})
		},
	})

	// 链路追踪：先生成/透传 request id，再为请求建立根 span，后续中间件与业务均可读取
	server.Use(httpx.WithRequestID())
	server.Use(middleware.Trace())
	// CORS：从配置读取允许的来源
	allowOrigins := cfg.CORS.AllowOrigins
	if len(allowOrigins) == 0 {
		allowOrigins = []string{"*"}
	}
	server.Use(httpx.WithCors(allowOrigins...))
	server.Use(httpx.WithRecovery())
	server.Use(httpx.WithLogger())
	// 日志中间件跳过：健康检查、日志查询、relay 转发路由（避免记录大请求体/敏感对话）
	server.Use(middleware.Log(logLogic, []string{"/health", "/api/v1/sys/logs", "/v1/"}, []string{"OPTIONS", "HEAD"}))

	authMw := middleware.Auth(j)
	loadAccountMw := middleware.LoadAccount(authSvc, cfg.App.AdminRoleID)

	v1 := server.Group("/api/v1")

	v1.AddRoute(httpx.Route{Method: "POST", Path: "/auth/login", Handler: authHandler.Login})
	v1.AddRoute(httpx.Route{Method: "POST", Path: "/auth/refresh", Handler: authHandler.Refresh})

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

	auth.AddRoutes([]httpx.Route{
		{Method: "GET", Path: "/dashboard/stats", Handler: dashboardHandler.Stats},
	})

	auth.AddRoutes([]httpx.Route{
		{Method: "GET", Path: "/billing/orders", Handler: billingHandler.ListOrders},
		{Method: "POST", Path: "/billing/orders", Handler: billingHandler.CreateOrder},
		{Method: "POST", Path: "/billing/orders/{id}/confirm", Handler: billingHandler.ConfirmOrder},
		{Method: "POST", Path: "/billing/orders/{id}/cancel", Handler: billingHandler.CancelOrder},
		{Method: "GET", Path: "/billing/transactions", Handler: billingHandler.ListTransactions},
		{Method: "POST", Path: "/billing/balance/adjust", Handler: billingHandler.AdjustBalance},
	})

	auth.AddRoutes([]httpx.Route{
		{Method: "GET", Path: "/llm/providers", Handler: providerHandler.List},
		{Method: "GET", Path: "/llm/providers/all", Handler: providerHandler.AllList},
		{Method: "GET", Path: "/llm/providers/{id}", Handler: providerHandler.Detail},
		{Method: "POST", Path: "/llm/providers", Handler: providerHandler.Create},
		{Method: "PUT", Path: "/llm/providers/{id}", Handler: providerHandler.Update},
		{Method: "DELETE", Path: "/llm/providers/{id}", Handler: providerHandler.Delete},
		{Method: "GET", Path: "/llm/providers/{id}/sync-models/preview", Handler: providerHandler.PreviewSyncModels},
		{Method: "POST", Path: "/llm/providers/{id}/sync-models", Handler: providerHandler.SyncModels},
	})

	auth.AddRoutes([]httpx.Route{
		{Method: "GET", Path: "/llm/models", Handler: modelHandler.List},
		{Method: "GET", Path: "/llm/models/all", Handler: modelHandler.AllList},
		{Method: "GET", Path: "/llm/models/{id}", Handler: modelHandler.Detail},
		{Method: "POST", Path: "/llm/models", Handler: modelHandler.Create},
		{Method: "PUT", Path: "/llm/models/{id}", Handler: modelHandler.Update},
		{Method: "DELETE", Path: "/llm/models/{id}", Handler: modelHandler.Delete},
	})

	auth.AddRoutes([]httpx.Route{
		{Method: "GET", Path: "/llm/tokens", Handler: tokenHandler.List},
		{Method: "GET", Path: "/llm/tokens/{id}", Handler: tokenHandler.Detail},
		{Method: "POST", Path: "/llm/tokens", Handler: tokenHandler.Create},
		{Method: "PUT", Path: "/llm/tokens/{id}", Handler: tokenHandler.Update},
		{Method: "DELETE", Path: "/llm/tokens/{id}", Handler: tokenHandler.Delete},
		{Method: "GET", Path: "/llm/tokens/{id}/reveal", Handler: tokenHandler.Reveal},
	})

	auth.AddRoutes([]httpx.Route{
		{Method: "GET", Path: "/llm/usage", Handler: usageHandler.List},
		{Method: "GET", Path: "/llm/usage/stats", Handler: usageHandler.Stats},
	})

	relayGroup := server.Group("/v1")
	relayGroup.AddRoutes([]httpx.Route{
		{Method: "POST", Path: "/chat/completions", Handler: relayHandler.ChatCompletions},
		{Method: "POST", Path: "/embeddings", Handler: relayHandler.Embeddings},
		{Method: "GET", Path: "/models", Handler: relayHandler.ListModels},
	})
}
