package main

import (
	"chihqiang/llm-gate/config"
	"chihqiang/llm-gate/db"
	"chihqiang/llm-gate/handler"
	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/route"

	"github.com/chihqiang/infra-go/conf"
	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/jwt"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/orm"
)

func main() {
	// 加载配置
	var cfg config.Config
	conf.MustLoad("config.yaml", &cfg)

	// 初始化日志
	log := logger.New(cfg.Logger)
	defer log.Sync()

	// 初始化数据库
	gormDB, err := orm.New(cfg.DB)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 数据库迁移
	if err := db.Migrate(gormDB); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 初始化 JWT
	j, err := jwt.New(cfg.JWT)
	if err != nil {
		log.Fatalf("JWT 初始化失败: %v", err)
	}

	// 初始化服务
	authSvc := logic.NewAuthLogic(gormDB, j)
	accountSvc := logic.NewAccountLogic(gormDB)
	roleSvc := logic.NewRoleLogic(gormDB)
	menuSvc := logic.NewMenuLogic(gormDB)
	logSvc := logic.NewLogLogic(gormDB)

	// 初始化 Handler
	authHandler := handler.NewAuthHandler(authSvc)
	accountHandler := handler.NewAccountHandler(accountSvc)
	roleHandler := handler.NewRoleHandler(roleSvc)
	menuHandler := handler.NewMenuHandler(menuSvc)
	logHandler := handler.NewLogHandler(logSvc)

	// 创建 HTTP 服务器
	server := httpx.NewServer(cfg.Server)

	// 注册路由
	route.Register(server, j, authSvc, logSvc, authHandler, accountHandler, roleHandler, menuHandler, logHandler)

	// 打印路由
	server.PrintRoutes()

	// 启动服务
	logger.Infof("服务启动 %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := server.Start(); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
