package config

import (
	"github.com/chihqiang/infra-go/httpx"
	"github.com/chihqiang/infra-go/jwt"
	"github.com/chihqiang/infra-go/logger"
	"github.com/chihqiang/infra-go/orm"
)

type Config struct {
	App    App                `json:"app"`
	Server httpx.ServerConfig `json:"server"`
	DB     orm.Config         `json:"db"`
	JWT    jwt.Config         `json:"jwt"`
	Logger logger.Config      `json:"logger"`
}

type App struct {
	Name    string `json:",default=llm-gate"`
	Version string `json:",default=0.0.1"`
}
