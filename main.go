package main

import (
	"chihqiang/llm-gate/config"
	"chihqiang/llm-gate/db"
	"chihqiang/llm-gate/handler"
	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/relay"
	"chihqiang/llm-gate/route"

	"github.com/chihqiang/infra-go/conf"
	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/jwt"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/orm"
)

func main() {
	var cfg config.Config
	conf.MustLoad("config.yaml", &cfg)

	log := logger.New(cfg.Logger)
	defer log.Sync()

	gormDB, err := orm.New(cfg.DB)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	if err := db.Migrate(gormDB); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	j, err := jwt.New(cfg.JWT)
	if err != nil {
		log.Fatalf("JWT 初始化失败: %v", err)
	}

	authSvc := logic.NewAuthLogic(gormDB, j)
	accountSvc := logic.NewAccountLogic(gormDB)
	roleSvc := logic.NewRoleLogic(gormDB)
	menuSvc := logic.NewMenuLogic(gormDB)
	logSvc := logic.NewLogLogic(gormDB)

	providerSvc := logic.NewProviderLogic(gormDB)
	modelSvc := logic.NewModelLogic(gormDB)
	tokenSvc := logic.NewTokenLogic(gormDB)
	usageSvc := logic.NewUsageLogic(gormDB)
	dashboardSvc := logic.NewDashboardLogic(gormDB)

	authHandler := handler.NewAuthHandler(authSvc)
	accountHandler := handler.NewAccountHandler(accountSvc)
	roleHandler := handler.NewRoleHandler(roleSvc)
	menuHandler := handler.NewMenuHandler(menuSvc)
	logHandler := handler.NewLogHandler(logSvc)
	dashboardHandler := handler.NewDashboardHandler(dashboardSvc)

	providerHandler := handler.NewProviderHandler(providerSvc)
	modelHandler := handler.NewModelHandler(modelSvc)
	tokenHandler := handler.NewTokenHandler(tokenSvc)
	usageHandler := handler.NewUsageHandler(usageSvc)
	relayHandler := relay.NewRelayHandler(gormDB, cfg.Relay)

	server := httpx.NewServer(cfg.Server)

	route.Register(server, j, authSvc, logSvc,
		authHandler, accountHandler, roleHandler, menuHandler, logHandler, dashboardHandler,
		providerHandler, modelHandler, tokenHandler, usageHandler, relayHandler)

	server.PrintRoutes()

	logger.Infof("服务启动 %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := server.Start(); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
