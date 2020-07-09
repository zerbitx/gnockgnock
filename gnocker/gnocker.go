package gnocker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber"
	"github.com/sirupsen/logrus"
	"github.com/zerbitx/gnockgnock/spec"
	"gopkg.in/yaml.v2"
)

type (
	fiberBinding func(string, ...fiber.Handler) *fiber.Route

	gnocker struct {
		app            *fiber.App
		configBasePath string
		handlers       map[string]map[string]map[string]fiber.Handler
		configs        map[string]struct{}
		handlerBases   map[string]fiberBinding
		pathsSeen      map[string]bool
		logger         logrus.FieldLogger
		port           int
		host           string
	}

	configConflict string

	config struct {
		port           int
		configBasePath string
		host           string
		logger         logrus.FieldLogger
	}

	// Option is a function that can modify a default config
	Option func(c *config)
)

// GnockerHeader is the constant for the head you pass if you need to overload paths/methods
const GnockerHeader = "X-GNOCKER"

// Error implements the error interface
func (cc configConflict) Error() string {
	return fmt.Sprintf("a config with the name %s already exists", string(cc))
}

// New returns a new gnocker with a default setup of up app and config on 127.0.0.1 on ports 8080 & 8081
func New(options ...Option) *gnocker {
	logrus.SetReportCaller(true)
	c := &config{
		port:           8080,
		logger:         logrus.StandardLogger(),
		host:           "127.0.0.1",
		configBasePath: "/gnockconfig",
	}

	for _, applyOption := range options {
		applyOption(c)
	}

	app := fiber.New(&fiber.Settings{
		ServerHeader:          "GnockGnock",
		DisableStartupMessage: true,
	})

	g := &gnocker{
		logger:         c.logger,
		app:            app,
		port:           c.port,
		host:           c.host,
		configBasePath: c.configBasePath,
		handlers:       map[string]map[string]map[string]fiber.Handler{},
		configs:        map[string]struct{}{},
		handlerBases: map[string]fiberBinding{
			http.MethodGet:     app.Get,
			http.MethodPost:    app.Post,
			http.MethodDelete:  app.Delete,
			http.MethodPatch:   app.Patch,
			http.MethodPut:     app.Put,
			http.MethodOptions: app.Options,
			http.MethodConnect: app.Connect,
			http.MethodTrace:   app.Trace,
			http.MethodHead:    app.Head,
		},
		pathsSeen: map[string]bool{},
	}

	g.initConfigEndpoints()

	return g
}

// Start starts both apps
func (g *gnocker) Start() error {
	errc := make(chan error)

	// Start up our main server
	go func() {
		g.logger.WithFields(logrus.Fields{"host": g.host, "port": g.port}).Info("main")
		errc <- g.app.Listen(fmt.Sprintf("%s:%d", g.host, g.port))
	}()

	return <-errc
}

// Shutdown gracefully shuts down both apps
func (g *gnocker) Shutdown() error {
	if shutdownErr := g.app.Shutdown(); shutdownErr != nil {
		return fmt.Errorf("failed to shutdown app %w", shutdownErr)
	}

	return nil
}

// WithLogger overrides the default logger
func WithLogger(l logrus.FieldLogger) Option {
	return func(c *config) {
		c.logger = l
	}
}

// WithHost sets the host
func WithHost(host string) Option {
	return func(c *config) {
		c.host = host
	}
}

// WithPort sets the main app's port
func WithPort(port int) Option {
	return func(c *config) {
		c.port = port
	}
}

// WithConfigBasePath sets the base path to post and look up configurations
func WithConfigBasePath(basePath string) Option {
	return func(c *config) {
		c.configBasePath = basePath
	}
}

// AddConfig will wire in a new configuration with its own set of routes and responses associated with a config name for
// header based differentiated access.
func (g *gnocker) AddConfig(operations spec.Configurations) error {
	for configName, operation := range operations {
		// Associate a config name first and key off that, rather than human readable config names?
		_, ok := g.handlers[configName]
		if ok {
			return configConflict(configName)
		}

		g.handlers[configName] = map[string]map[string]fiber.Handler{}

		if operation.TTL != "" {
			dur, err := time.ParseDuration(operation.TTL)

			if err != nil {
				g.logger.WithError(err).Error()
				return fmt.Errorf("failed to parse duration %s", operation.TTL)
			}

			go func() {
				<-time.After(dur)
				g.logger.WithField("config", configName).Info("Removing expired")
				delete(g.handlers, configName)
			}()
		}

		for path, methods := range operation.Paths {
			for m, options := range methods {
				method := strings.ToUpper(m)

				g.logger.WithFields(logrus.Fields{
					"config": configName,
					"path":   path,
					"method": method,
				}).Debug("wiring")

				if _, ok := g.handlers[configName][path]; !ok {
					g.handlers[configName][path] = map[string]fiber.Handler{}
				}

				var tpl *template.Template
				var err error
				if options.ResponseBodyTemplate != "" {
					tpl, err = template.New(configName).Parse(options.ResponseBodyTemplate)

					if err != nil {
						g.logger.
							WithError(err).
							WithField("template", options.ResponseBodyTemplate).
							Error("Failed to parse template string")
						return err
					}
				}

				g.handlers[configName][path][method] = func(c *fiber.Ctx) {
					c.Status(options.StatusCode)

					for _, headers := range options.ResponseHeaders {
						for header, value := range headers {
							c.Fasthttp.Response.Header.Add(header, value)
						}
					}

					// If a template was configured and parsed, correctly
					if tpl != nil {
						var buf bytes.Buffer
						templateVars := map[string]string{}

						// populate the template data from the params
						for _, name := range c.Route().Params {
							templateVars[name] = c.Params(name)
						}

						err := tpl.Execute(&buf, templateVars)

						if err != nil {
							g.logger.WithError(err).Error("failed to execute template")
							c.SendStatus(http.StatusInternalServerError)
						}

						c.SendBytes(buf.Bytes())
					} else if options.ResponseBody != "" {
						// otherwise use the static response
						c.Send(options.ResponseBody)
					}
				}

				if _, seen := g.pathsSeen[method+":"+path]; !seen {
					go func(path, method, configName string) {
						g.handlerBases[method](path, func(c *fiber.Ctx) {
							// If no config name was sent, serve the first path configured by this instance
							// otherwise look up the correct handler by the config name header sent.
							var handler map[string]fiber.Handler
							servingConfig := configName
							if configFromHeader := c.Get(GnockerHeader); configFromHeader != "" {
								servingConfig = configFromHeader
							}

							g.logger.WithFields(logrus.Fields{
								"config": servingConfig,
								"path":   c.Path(),
								"method": method,
							}).Debug("serving")

							handler = g.handlers[servingConfig][path]

							if handler != nil && handler[c.Method()] != nil {
								handler[c.Method()](c)
							} else {
								g.logger.WithError(err).Error("failed to find handler")
								c.SendStatus(http.StatusNotFound)
							}
						})
					}(path, method, configName)
					g.pathsSeen[method+":"+path] = true
				}
			}
		}
	}

	return nil
}

func (g *gnocker) initConfigEndpoints() {
	g.logger.
		WithFields(logrus.Fields{
			http.MethodPost: g.configBasePath,
			http.MethodGet:  g.configBasePath,
		}).Debug("config endpoints")

	g.app.Post(g.configBasePath, func(c *fiber.Ctx) {
		bodyReader := strings.NewReader(c.Body())

		newOperations := spec.Configurations{}
		err := yaml.NewDecoder(bodyReader).Decode(newOperations)

		if err != nil {
			g.logger.WithError(err).Error("failed to decode yaml")
			c.SendStatus(http.StatusBadRequest)
			return
		}

		err = g.AddConfig(newOperations)

		if err != nil {
			g.logger.WithError(err).Error("failed to add request")
			c.SendStatus(http.StatusBadRequest)
			return
		}

		var configNames []string
		for name := range newOperations {
			g.configs[name] = struct{}{}
			configNames = append(configNames, name)
		}

		c.Status(http.StatusCreated)

		encoder := json.NewEncoder(c.Fasthttp.Response.BodyWriter())
		encoder.SetIndent("", " ")
		err = encoder.Encode(configNames)

		if err != nil {
			g.logger.WithError(err).Error("Failed to encode response")
			c.SendStatus(http.StatusInternalServerError)
			return
		}
	})

	g.app.Get(g.configBasePath, func(c *fiber.Ctx) {
		var configs []string
		for config := range g.configs {
			configs = append(configs, config)
		}

		encoder := json.NewEncoder(c.Fasthttp.Response.BodyWriter())
		encoder.SetIndent("", " ")
		err := encoder.Encode(configs)

		if err != nil {
			g.logger.WithError(err).Error("Failed to encode response")
			c.SendStatus(http.StatusInternalServerError)
			return
		}
	})
}
