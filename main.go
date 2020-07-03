package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber"
	"github.com/zerbitx/gnockgnock/config"
	"github.com/zerbitx/gnockgnock/gnocker"
	"github.com/zerbitx/gnockgnock/spec"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

func main() {
	cfg := config.New()

	fiberSettings := &fiber.Settings{
		ServerHeader:          "GnockGnock",
		DisableStartupMessage: true,
	}
	app := fiber.New(fiberSettings)
	configApp := fiber.New(fiberSettings)
	logger := zap.NewExample().Sugar()
	defer logger.Sync()
	g := gnocker.New(app, configApp, logger)

	// Try to load a default config
	{
		configYaml, err := os.OpenFile(cfg.ConfigFilePath, os.O_RDONLY, 0644)

		// No config...no problem
		if err == nil {
			operations := spec.Configurations{}
			if err := yaml.NewDecoder(configYaml).Decode(operations); err != nil {
				log.Fatalf("Failed to decode yaml: %operations", err)
			}

			if err := g.AddConfig(operations); err != nil {
				log.Fatalf("failed to setup initial config: %s", err)
			}
		}
	}

	errc := make(chan error)

	// Start up our main server
	go func() {
		logger.Infow("main", "host", cfg.Host, "port", cfg.AppPort)
		errc <- app.Listen(fmt.Sprintf("%s:%d", cfg.Host, cfg.AppPort))
	}()

	// Start up the config server
	go func() {
		logger.Infow("config", "host", cfg.Host, "port", cfg.ConfigPort)
		errc <- configApp.Listen(fmt.Sprintf("%s:%d", cfg.Host, cfg.ConfigPort))
	}()

	fmt.Println(<-errc)
}
