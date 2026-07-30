package config

import (
	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/jwt"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/orm"
	"github.com/chihqiang/infra-go/redisx"
)

type Config struct {
	App    App                `json:"app"`
	Server httpx.ServerConfig `json:"server"`
	DB     orm.Config         `json:"db"`
	JWT    jwt.Config         `json:"jwt"`
	Logger logger.Config      `json:"logger"`
	Relay  RelayConfig        `json:"relay"`
	CORS   CORSConfig         `json:"cors"`
	Pprof  PprofConfig        `json:"pprof"`
	Redis  redisx.Config      `json:"redis,optional"`
}

type RelayConfig struct {
	Timeout         int             `json:",default=120"`
	MaxBodyMB       int             `json:",default=32"`
	PreConsumeQuota int64           `json:",default=1000"`
	RateLimit       RateLimitConfig `json:"rate_limit,optional"`
	Upstream        UpstreamConfig  `json:"upstream,optional"`
}

type RateLimitConfig struct {
	Enabled bool    `json:",optional"`
	Rate    float64 `json:",default=10"`
	Burst   float64 `json:",default=20"`
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
