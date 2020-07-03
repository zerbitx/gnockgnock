package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber"
	"github.com/zerbitx/gnockgnock/config"
	"github.com/zerbitx/gnockgnock/gnocker"
	"github.com/zerbitx/gnockgnock/spec"
	"gopkg.in/yaml.v2"
)

func main() {
	cfg := config.New()

	app := fiber.New()
	configApp := fiber.New()
	h := gnocker.New(app, configApp)

	// Try to load a default config
	{
		configYaml, err := os.OpenFile(cfg.ConfigFilePath, os.O_RDONLY, 0644)

		// No config...no problem
		if err == nil {
			operations := spec.Configurations{}
			if err := yaml.NewDecoder(configYaml).Decode(operations); err != nil {
				log.Fatalf("Failed to decode yaml: %operations", err)
			}

			if err := h.AddConfig(operations); err != nil {
				log.Fatalf("failed to setup initial config: %s", err)
			}
		}
	}

	errc := make(chan error)

	go func() {
		errc <- app.Listen(fmt.Sprintf("%s:%d", cfg.Host, cfg.AppPort))
	}()

	go func() {
		errc <- configApp.Listen(fmt.Sprintf("%s:%d", cfg.Host, cfg.ConfigPort))
	}()

	fmt.Println(<-errc)
}
