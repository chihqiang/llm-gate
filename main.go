package main

import (
	"context"
	"os"
	"time"

	"chihqiang/llm-gate/cache"
	"chihqiang/llm-gate/config"
	"chihqiang/llm-gate/db"
	"chihqiang/llm-gate/handler"
	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/relay"
	"chihqiang/llm-gate/route"
	"chihqiang/llm-gate/security"

	"github.com/chihqiang/infra-go/conf"
	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/jwt"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/orm"
	"github.com/chihqiang/infra-go/redisx"
	"gorm.io/gorm"
)

func main() {
	var cfg config.Config
	conf.MustLoad("config.yaml", &cfg)

	// 支持通过环境变量覆盖 JWT Secret / 加密密钥
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWT.Secret = secret
	}
	if key := os.Getenv("ENCRYPT_KEY"); key != "" {
		cfg.Security.EncryptKey = key
	}

	log := logger.New(cfg.Logger)
	defer log.Sync()
	logger.SetGlobal(log)

	// 密钥加密器：security.encrypt_key 或 JWT Secret 派生
	cipher, err := security.New(cfg.Security.EncryptKey, cfg.JWT.Secret)
	if err != nil {
		log.Fatalf("加密器初始化失败: %v", err)
	}

	gormDB, err := orm.New(cfg.DB)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Fatalf("获取 sql.DB 失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(20)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// SQLite：设置 busy_timeout 避免写冲突时立即报错
	if cfg.DB.Driver == "sqlite" {
		gormDB.Exec("PRAGMA busy_timeout = 5000")
	}

	// 配置 GORM 会话：预编译语句 + 默认超时
	gormDB = gormDB.Session(&gorm.Session{PrepareStmt: true})

	if err := db.Migrate(gormDB); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	j, err := jwt.New(cfg.JWT)
	if err != nil {
		log.Fatalf("JWT 初始化失败: %v", err)
	}

	// 初始化缓存：Redis 配置不为空时使用 Redis，否则使用内存缓存。
	var authCache, providerCache, modelListCache, accountCache cache.Cache
	if cfg.Redis.Addr != "" {
		redisClient, err := redisx.New(cfg.Redis)
		if err != nil {
			log.Fatalf("Redis 连接失败: %v", err)
		}
		accountCache = cache.NewRedis(redisClient)
		authCache = cache.NewRedis(redisClient)
		providerCache = cache.NewRedis(redisClient)
		modelListCache = cache.NewRedis(redisClient)
	} else {
		accountCache = cache.NewMemory()
		authCache = cache.NewMemory()
		providerCache = cache.NewMemory()
		modelListCache = cache.NewMemory()
	}

	authSvc := logic.NewAuthLogic(gormDB, j, accountCache)
	accountSvc := logic.NewAccountLogic(gormDB, accountCache)
	roleSvc := logic.NewRoleLogic(gormDB, cfg.App.AdminRoleID)
	menuSvc := logic.NewMenuLogic(gormDB)
	logSvc := logic.NewLogLogic(gormDB)

	providerSvc := logic.NewProviderLogic(gormDB, providerCache, cipher)
	modelSvc := logic.NewModelLogic(gormDB, modelListCache)
	tokenSvc := logic.NewTokenLogic(gormDB, authCache, cipher)
	usageSvc := logic.NewUsageLogic(gormDB)
	dashboardSvc := logic.NewDashboardLogic(gormDB)
	billingSvc := logic.NewBillingLogic(gormDB)

	authHandler := handler.NewAuthHandler(authSvc)
	accountHandler := handler.NewAccountHandler(accountSvc)
	roleHandler := handler.NewRoleHandler(roleSvc)
	menuHandler := handler.NewMenuHandler(menuSvc)
	logHandler := handler.NewLogHandler(logSvc)
	dashboardHandler := handler.NewDashboardHandler(dashboardSvc)
	billingHandler := handler.NewBillingHandler(billingSvc)

	providerHandler := handler.NewProviderHandler(providerSvc)
	modelHandler := handler.NewModelHandler(modelSvc)
	tokenHandler := handler.NewTokenHandler(tokenSvc)
	usageHandler := handler.NewUsageHandler(usageSvc)
	relayHandler := relay.NewRelayHandler(gormDB, cfg, cipher, authCache, providerCache, modelListCache)
	defer relayHandler.Stop()

	// 数据保留清理任务
	retentionCtx, cancelRetention := context.WithCancel(context.Background())
	defer cancelRetention()
	logic.NewRetentionCleaner(gormDB, cfg.Retention).Start(retentionCtx)

	server := httpx.NewServer(cfg.Server)

	route.Register(server, j, authSvc, logSvc, cfg,
		authHandler, accountHandler, roleHandler, menuHandler, logHandler, dashboardHandler, billingHandler,
		providerHandler, modelHandler, tokenHandler, usageHandler, relayHandler)

	server.PrintRoutes()

	logger.Infof("服务启动 %s:%d", cfg.Server.Host, cfg.Server.Port)
	if err := server.Start(); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
