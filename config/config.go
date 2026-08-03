package config

import (
	"time"

	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/jwt"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/orm"
	"github.com/chihqiang/infra-go/redisx"
)

type Config struct {
	App       App                `json:"app"`
	Server    httpx.ServerConfig `json:"server"`
	DB        orm.Config         `json:"db"`
	JWT       jwt.Config         `json:"jwt"`
	Logger    logger.Config      `json:"logger"`
	Relay     RelayConfig        `json:"relay"`
	Billing   BillingConfig      `json:"billing"`
	Security  SecurityConfig     `json:"security"`
	Retention RetentionConfig    `json:"retention"`
	Alert     AlertConfig        `json:"alert"`
	CORS      CORSConfig         `json:"cors"`
	Pprof     PprofConfig        `json:"pprof"`
	Redis     redisx.Config      `json:"redis,optional"`
}

type RelayConfig struct {
	Timeout          int             `json:",default=120"`
	MaxBodyMB        int             `json:",default=32"`
	PreConsumeCents  int64           `json:",default=100"`
	StreamFallbackCents int64        `json:",default=100"`
	RateLimit        RateLimitConfig `json:"rate_limit,optional"`
	Upstream         UpstreamConfig  `json:"upstream,optional"`
	Failover         FailoverConfig  `json:"failover,optional"`
}

// BillingConfig 真实货币计费配置。金额单位均为分（1 元 = 100 分）。
type BillingConfig struct {
	// BasePriceCentsPer1K 每千 token 基础单价（分）。
	// 单次请求费用 = (prompt/1000*ratio + completion/1000*ratio*completion_ratio) * base_price_cents_per_1k
	BasePriceCentsPer1K int64 `json:",default=2"`
	// MinBalanceCents 余额低于该值时触发告警（分）。
	MinBalanceCents int64 `json:",default=1000"`
}

// SecurityConfig 安全配置。
type SecurityConfig struct {
	// EncryptKey AES-256 加密密钥，64 位十六进制（32 字节）。
	// 未配置时回退到 JWT Secret 派生（生产环境必须配置）。
	EncryptKey string `json:",optional"`
	// RevealAudit 记录密钥查看操作日志。
	RevealAudit bool `json:",default=true"`
}

// RetentionConfig 数据保留与清理策略。
type RetentionConfig struct {
	// UsageDays 用量日志保留天数，<=0 不清理。
	UsageDays int `json:",default=0"`
	// TransactionDays 余额流水保留天数，<=0 不清理。
	TransactionDays int `json:",default=0"`
	// ExpiredTokenCleanup 清理已过期且超过该天数（距过期时间）的 Token，<=0 不清理。
	ExpiredTokenGraceDays int `json:",default=0"`
	// CheckInterval 清理检查间隔。
	CheckInterval time.Duration `json:",default=24h"`
}

// AlertConfig 告警通知。
type AlertConfig struct {
	// Enabled 是否启用 webhook 告警。
	Enabled bool `json:",optional"`
	// WebhookURL 告警回调地址。
	WebhookURL string `json:",optional"`
	// SignSecret 签名密钥（用于 HMAC-SHA256 签名），留空不签名。
	SignSecret string `json:",optional"`
	// Cooldown 同类告警最小间隔，防止轰炸。
	Cooldown time.Duration `json:",default=10m"`
}

type RateLimitConfig struct {
	Enabled bool    `json:",optional"`
	Rate    float64 `json:",default=10"`
	Burst   float64 `json:",default=20"`
	// GlobalRate 全局限流（所有请求），>0 启用。
	GlobalRate float64 `json:",default=0"`
	GlobalBurst float64 `json:",default=0"`
	// AccountRate 单账户限流，>0 启用。
	AccountRate float64 `json:",default=0"`
	AccountBurst float64 `json:",default=0"`
}

type FailoverConfig struct {
	// Enabled 是否启用同模型多服务商自动降级。
	Enabled bool `json:",default=true"`
	// FailureThreshold 窗口内连续失败阈值，达到后熔断。
	FailureThreshold int `json:",default=5"`
	// Window 统计窗口。
	Window time.Duration `json:",default=60s"`
	// Cooldown 熔断持续时间。
	Cooldown time.Duration `json:",default=30s"`
	// HealthProbeEnabled 是否周期性探测上游 /models 健康。
	HealthProbeEnabled bool `json:",optional"`
	// HealthProbeInterval 健康探测间隔。
	HealthProbeInterval time.Duration `json:",default=30s"`
}

type UpstreamConfig struct {
	RetryEnabled bool  `json:",optional"`
	MaxRetries   int   `json:",default=2"`
	RetryDelayMs int64 `json:",default=200"`
}

type CORSConfig struct {
	AllowOrigins []string `json:",optional"`
}

type PprofConfig struct {
	Enabled bool `json:",default=false"`
}

type App struct {
	Name        string `json:",default=llm-gate"`
	Version     string `json:",default=0.0.1"`
	AdminRoleID int64  `json:",default=1"`
}
