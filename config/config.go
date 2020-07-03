package config

import (
	"github.com/kelseyhightower/envconfig"
)

type (
	Env struct {
		Host           string `envconfig:"HOST" default:"127.0.0.1"`
		AppPort        int    `envconfig:"PORT" default:"8080"`
		ConfigPort     int    `envconfig:"CONFIG_PORT" default:"8081"`
		ConfigFilePath string `envconfig:"GNOCK_CONFIG" default:"./gnockgnock.yaml"`
	}
)

func New() *Env {
	cfg := &Env{}

	envconfig.MustProcess("", cfg)

	return cfg
}
