package config

import (
	"github.com/kelseyhightower/envconfig"
)

type (
	Env struct {
		Host           string `envconfig:"HOST" default:"127.0.0.1"`
		Port           int    `envconfig:"PORT" default:"8080"`
		ConfigFilePath string `envconfig:"GNOCK_CONFIG" default:"./gnockgnock.yaml"`
		LogLevel       string `envconfig:"LOG_LEVEL" default:"debug"`
	}
)

func New() *Env {
	cfg := &Env{}

	envconfig.MustProcess("", cfg)

	return cfg
}
