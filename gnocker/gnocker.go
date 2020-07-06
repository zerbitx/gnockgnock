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
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/zerbitx/gnockgnock/spec"
	"gopkg.in/yaml.v2"
)

type (
	fiberBinding func(string, ...fiber.Handler) *fiber.Route

	gnocker struct {
		app          *fiber.App
		configApp    *fiber.App
		handlers     map[string]map[string]map[string]fiber.Handler
		tokens       map[string]string
		handlerBases map[string]fiberBinding
		pathsSeen    map[string]bool
		logger       logrus.FieldLogger
		port         int
		host         string
	}

	configConflict string

	config struct {
		port       int
		configPort int
		host       string
		logger     logrus.FieldLogger
	}

	Option func(c *config)
)

// TokenHeader is the constant for the head you pass if you need to overload paths/methods
const TokenHeader = "X-GNOCKER"

// Error implements the error interface
func (cc configConflict) Error() string {
	return fmt.Sprintf("a config with the name %s already exists", cc)
}

// New returns a new gnocker with a default setup of up app and config on 127.0.0.1 on ports 8080 & 8081
func New(options ...Option) *gnocker {
	logrus.SetReportCaller(true)
	c := &config{
		port:   8080,
		logger: logrus.StandardLogger(),
		host:   "127.0.0.1",
	}

	for _, applyOption := range options {
		applyOption(c)
	}

	// Silly convention?
	if c.configPort == 0 {
		c.configPort = c.port + 1
	}

	fiberSettings := &fiber.Settings{
		ServerHeader:          "GnockGnock",
		DisableStartupMessage: true,
	}

	app := fiber.New(fiberSettings)
	configApp := fiber.New(fiberSettings)

	g := &gnocker{
		logger:    c.logger,
		app:       app,
		configApp: configApp,
		port:      c.port,
		host:      c.host,
		handlers:  map[string]map[string]map[string]fiber.Handler{},
		tokens:    map[string]string{},
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

	g.initConfigApp(configApp)

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

	// Start up the config server
	go func() {
		configPort := g.port + 1
		g.logger.WithFields(logrus.Fields{"host": g.host, "port": configPort}).Info("config")
		errc <- g.configApp.Listen(fmt.Sprintf("%s:%d", g.host, configPort))
	}()

	return <-errc
}

// Shutdown gracefully shuts down both apps
func (g *gnocker) Shutdown() error {
	var err error = nil

	if shutdownErr := g.configApp.Shutdown(); shutdownErr != nil {
		err = fmt.Errorf("failed to shutdown config app %w", err)
	}

	if shutdownErr := g.app.Shutdown(); shutdownErr != nil {
		err = fmt.Errorf("failed to shutdown app %w", err)
	}

	return err
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

// WithConfigPort sets the config app's port
func WithConfigPort(configPort int) Option {
	return func(c *config) {
		c.configPort = configPort
	}
}

// AddConfig will wire in a new configuration with its own set of routes and responses associated with a token for
// header based differentiated access.
func (g *gnocker) AddConfig(operations spec.Configurations) error {
	for configName, operation := range operations {
		// Associate a token first and key off that, rather than human readable config names?
		_, ok := g.handlers[configName]
		if ok {
			return configConflict(configName)
		}

		g.handlers[configName] = map[string]map[string]fiber.Handler{}

		if operation.TTL != "" {
			dur, err := time.ParseDuration(operation.TTL)

			if err != nil {
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
						return err
					}
				}

				g.handlers[configName][path][method] = func(c *fiber.Ctx) {
					c.Status(options.StatusCode)

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
							gnockerToken := c.Get(TokenHeader)

							// If no token was sent, serve the first path configured by this instance
							// otherwise look up the correct handler by the token sent
							var handler map[string]fiber.Handler
							servingConfig := configName
							if gnockerToken != "" {
								servingConfig = g.tokens[gnockerToken]
							}

							g.logger.WithFields(logrus.Fields{
								"config": servingConfig,
								"token":  gnockerToken,
								"path":   c.Path(),
								"method": method,
							}).Debug("serving")

							handler = g.handlers[servingConfig][path]

							if handler != nil {
								handler[c.Method()](c)
							} else {
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

func (g *gnocker) initConfigApp(configApp *fiber.App) {
	configApp.Post("/config", func(c *fiber.Ctx) {
		bodyReader := strings.NewReader(c.Body())

		newOperations := spec.Configurations{}
		err := yaml.NewDecoder(bodyReader).Decode(newOperations)

		if err != nil {
			c.SendStatus(http.StatusBadRequest)
			return
		}

		err = g.AddConfig(newOperations)

		if err != nil {
			c.SendStatus(http.StatusBadRequest)
			return
		}

		newTokens := map[string]string{}
		for name := range newOperations {
			gnockerToken := uuid.New().String()
			g.tokens[gnockerToken] = name
			newTokens[name] = gnockerToken
		}

		c.Status(http.StatusCreated)

		encoder := json.NewEncoder(c.Fasthttp.Response.BodyWriter())
		encoder.SetIndent("", " ")
		err = encoder.Encode(newTokens)

		if err != nil {
			c.SendStatus(http.StatusInternalServerError)
			return
		}
	})

	configApp.Get("/tokens", func(c *fiber.Ctx) {
		tokens := map[string]string{}
		for token, config := range g.tokens {
			tokens[config] = token
		}

		encoder := json.NewEncoder(c.Fasthttp.Response.BodyWriter())
		encoder.SetIndent("", " ")
		err := encoder.Encode(tokens)

		if err != nil {
			c.SendStatus(http.StatusInternalServerError)
			return
		}
	})
}
