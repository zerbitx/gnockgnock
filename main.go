package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/sirupsen/logrus"
	"github.com/zerbitx/gnockgnock/config"
	"github.com/zerbitx/gnockgnock/gnocker"
	"github.com/zerbitx/gnockgnock/spec"
	"gopkg.in/yaml.v2"
)

func main() {
	cfg := config.New()

	var logger logrus.FieldLogger = logrus.StandardLogger().WithField("gnock", "gnock")
	setLogLevel(cfg.LogLevel)
	logrus.SetReportCaller(true)

	g := gnocker.New(
		gnocker.WithHost(cfg.Host),
		gnocker.WithPort(cfg.Port),
		gnocker.WithConfigPort(cfg.ConfigPort),
		gnocker.WithLogger(logger))

	go captureInterrupt(g.Shutdown)

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

	fmt.Println("Servers shutdown due to: ", g.Start())
}

func setLogLevel(lvlStr string) {
	lvl, err := logrus.ParseLevel(lvlStr)

	if err == nil {
		logrus.SetLevel(lvl)
		return
	}

	logrus.SetLevel(logrus.WarnLevel)
}

func captureInterrupt(shutdown func() error) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)

	<-c
	err := shutdown()

	if err != nil {
		fmt.Println("Something went wrong during shutdown: ", err)
	}
}
