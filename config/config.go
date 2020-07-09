package config

import (
	"github.com/kelseyhightower/envconfig"
)

type (
	// Env holds the values of environment variable based configuration
	Env struct {
		Host           string `envconfig:"HOST" default:"127.0.0.1"`
		Port           int    `envconfig:"PORT" default:"8080"`
		ConfigPort     int    `envconfig:"CONFIG_PORT" default:"8081"`
		ConfigFilePath string `envconfig:"GNOCK_CONFIG" default:"./gnockgnock.yaml"`
		ConfigBasePath string `envconfig:"GNOCK_BASE_PATH" default:"/gnockconfig"`
		LogLevel       string `envconfig:"LOG_LEVEL" default:"debug"`
	}
)

// New returns a new Env config
func New() *Env {
	cfg := &Env{}

	envconfig.MustProcess("", cfg)

	return cfg
}
