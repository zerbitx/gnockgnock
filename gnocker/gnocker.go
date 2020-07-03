package gnocker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber"
	"github.com/google/uuid"
	"github.com/zerbitx/gnockgnock/spec"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type (
	fiberBinding func(string, ...fiber.Handler) *fiber.Route

	gnocker struct {
		app          *fiber.App
		handlers     map[string]map[string]map[string]fiber.Handler
		tokens       map[string]string
		handlerBases map[string]fiberBinding
		pathsSeen    map[string]bool
		logger       *zap.SugaredLogger
	}

	configConflict string

	tokensResponse struct {
		ConfigName string `json:"configName"`
		Token      string `json:"token"`
	}
)

const TOKEN_HEADER = "X-GNOCKER"

func (cc configConflict) Error() string {
	return fmt.Sprintf("a config with the name %s already exists", cc)
}

func New(app *fiber.App, configApp *fiber.App, logger *zap.SugaredLogger) *gnocker {
	h := &gnocker{
		handlers: map[string]map[string]map[string]fiber.Handler{},
		tokens:   map[string]string{},
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
		logger:    logger,
	}

	h.initConfigApp(configApp)

	return h
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
				g.logger.Infow("Removing expired",
					"config", configName)
				delete(g.handlers, configName)
			}()
		}

		for path, methods := range operation.Paths {
			for m, options := range methods {
				method := strings.ToUpper(m)

				g.logger.Debugw("wiring",
					"config", configName,
					"path", path,
					"method", method)

				if _, ok := g.handlers[configName][path]; !ok {
					g.handlers[configName][path] = map[string]fiber.Handler{}
				}
				// create a handler to spec
				g.handlers[configName][path][method] = func(c *fiber.Ctx) {
					c.Status(options.StatusCode)
					if options.Payload != "" {
						c.Send(options.Payload)
					}
				}

				if _, seen := g.pathsSeen[method+":"+path]; !seen {
					g.handlerBases[method](path, func(c *fiber.Ctx) {
						gnockerToken := c.Get(TOKEN_HEADER)

						// If no token was sent, serve the first path configured by this instance
						// otherwise look up the correct handler by the token sent
						var handler map[string]fiber.Handler
						servingConfig := configName
						if gnockerToken != "" {
							servingConfig = g.tokens[gnockerToken]
						}

						g.logger.Debugw("serving",
							"config", servingConfig,
							"token", gnockerToken,
							"path", c.Path(),
							"method", method)

						handler = g.handlers[servingConfig][c.Path()]

						if handler != nil {
							handler[method](c)
						} else {
							c.SendStatus(http.StatusNotFound)
						}
					})
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

		gnockerToken := uuid.New().String()
		for name := range newOperations {
			g.tokens[gnockerToken] = name
		}

		c.Status(http.StatusAccepted)
		c.Send(fmt.Sprintf("Send the X-GNOCKER header with this token (%s) to invoke this configuration you can hit the config API /tokens to retrieve all currently configured tokens\n", gnockerToken))
	})

	configApp.Get("/tokens", func(c *fiber.Ctx) {
		var tokens []tokensResponse
		for token, config := range g.tokens {
			tokens = append(tokens, tokensResponse{
				ConfigName: config,
				Token:      token,
			})
		}

		var buf bytes.Buffer

		err := json.NewEncoder(&buf).Encode(tokens)

		if err != nil {
			c.SendStatus(http.StatusInternalServerError)
			return
		}

		c.SendBytes(buf.Bytes())
	})
}
