package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chihqiang/llm-gate/cache"
	"chihqiang/llm-gate/config"
	"chihqiang/llm-gate/db"
	"chihqiang/llm-gate/handler"
	"chihqiang/llm-gate/logic"
	"chihqiang/llm-gate/relay"
	"chihqiang/llm-gate/route"
	"chihqiang/llm-gate/security"

	"github.com/chihqiang/ten-go/core/conf"
	"github.com/chihqiang/ten-go/core/logger"
	"github.com/chihqiang/ten-go/core/orm"
	"github.com/chihqiang/ten-go/core/redisx"
	"github.com/chihqiang/ten-go/core/trace"
	"github.com/chihqiang/ten-go/rest"
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

	// 链路追踪：启动全局 TracerProvider（Endpoint 为空时 span 不导出，但 trace_id 仍会注入日志）
	trace.StartAgent(cfg.Trace)
	defer trace.StopAgent()

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

	tokens, err := logic.NewTokenManager(cfg.JWT)
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

	authSvc := logic.NewAuthLogic(gormDB, tokens, accountCache)
	accountSvc := logic.NewAccountLogic(gormDB, accountCache)
	roleSvc := logic.NewRoleLogic(gormDB, cfg.AdminRoleID)
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

	// ten-go rest 的恢复/请求 ID/访问日志默认值只在零值 Config 时启用；
	// 这里由配置文件构造 Config，显式打开这几项内置中间件。
	defaults := rest.DefaultConfig()
	cfg.Middlewares.Recovery = defaults.Middlewares.Recovery
	cfg.Middlewares.RequestID = defaults.Middlewares.RequestID
	logCfg := defaults.Middlewares.Log
	logCfg.ServiceName = cfg.Name
	cfg.Middlewares.Log = logCfg

	allowOrigins := cfg.CORS.AllowOrigins
	if len(allowOrigins) == 0 {
		allowOrigins = []string{"*"}
	}
	server, err := rest.NewServer(cfg.Config, rest.WithLogger(log), rest.WithCORS(allowOrigins...))
	if err != nil {
		log.Fatalf("创建 HTTP 服务失败: %v", err)
	}

	route.Register(server, tokens, authSvc, logSvc, cfg,
		authHandler, accountHandler, roleHandler, menuHandler, logHandler, dashboardHandler, billingHandler,
		providerHandler, modelHandler, tokenHandler, usageHandler, relayHandler)

	_ = server.PrintRoutes(os.Stdout)

	// ten-go rest.Server 不内置信号处理，这里自行监听并优雅关闭。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		shutdownTimeout := cfg.ShutdownTimeout
		if shutdownTimeout <= 0 {
			shutdownTimeout = 10 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Errorf("服务优雅关闭失败: %v", err)
		}
	}()

	logger.Infof("服务启动 %s:%d", cfg.Host, cfg.Port)
	if err := server.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("服务启动失败: %v", err)
	}
}
