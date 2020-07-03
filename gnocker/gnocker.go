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
	"gopkg.in/yaml.v2"
)

type (
	fiberBinding func(string, ...fiber.Handler) *fiber.Route

	gnocker struct {
		app      *fiber.App
		handlers map[string]map[string]fiber.Handler
		tokens   map[string]string
		handlerBases map[string]fiberBinding
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

func New(app *fiber.App, configApp *fiber.App) *gnocker {
	h := &gnocker{
		handlers: map[string]map[string]fiber.Handler{},
		tokens:   map[string]string{},
		handlerBases: map[string]fiberBinding{
			http.MethodGet: app.Get,
			http.MethodPost: app.Post,
			http.MethodDelete: app.Delete,
			http.MethodPatch: app.Patch,
			http.MethodPut: app.Put,
			http.MethodOptions: app.Options,
		},
	}

	h.startConfigApp(configApp)

	return h
}

// AddConfig will wire in a new configuration with its own set of routes and responses associated with a token for
// header based differentiated access.
func (h *gnocker) AddConfig(operations spec.Configurations) error {
	for configName, operation := range operations {
		// Associate a token first and key off that, rather than human readable config names?
		_, ok := h.handlers[configName]
		if ok {
			return configConflict(configName)
		}

		h.handlers[configName] = map[string]fiber.Handler{}
		
		if operation.TTL != "" {
			dur, err := time.ParseDuration(operation.TTL)
			
			if err != nil {
				return fmt.Errorf("failed to parse duration %s", operation.TTL)
			}
			
			go func() {
				<-time.After(dur)
				delete(h.handlers, configName)
			}()
		}
		
		for path, methods := range operation.Paths {
			for m, options := range methods {
				method := strings.ToUpper(m)
				
				// create a handler to spec
				h.handlers[configName][method] = func(c *fiber.Ctx) {
					c.Status(options.StatusCode)
					if options.Payload != "" {
						c.Send(options.Payload)
					}
				}
				
				h.handlerBases[method](path, func(c *fiber.Ctx) {
					gnockerToken := c.Get(TOKEN_HEADER)
					
					// If no token was sent, serve the first path configured by this instance
					// otherwise look up the correct handler by the token sent
					var handler map[string]fiber.Handler
					if gnockerToken != "" {
						cn := h.tokens[gnockerToken]
						
						handler = h.handlers[cn]
					} else {
						handler = h.handlers[configName]
					}
					
					if handler != nil {
						handler[method](c)
					} else {
						c.SendStatus(http.StatusNotFound)
					}
				})
			}
		}
	}

	return nil
}

func (h *gnocker) startConfigApp(configApp *fiber.App) {
	configApp.Post("/config", func(c *fiber.Ctx) {
		bodyReader := strings.NewReader(c.Body())

		newOperations := spec.Configurations{}
		err := yaml.NewDecoder(bodyReader).Decode(newOperations)

		if err != nil {
			c.SendStatus(http.StatusBadRequest)
			return
		}

		err = h.AddConfig(newOperations)

		if err != nil {
			c.SendStatus(http.StatusBadRequest)
			return
		}
		
		gnockerToken := uuid.New().String()
		for name := range newOperations {
			h.tokens[gnockerToken] = name
		}

		c.Status(http.StatusAccepted)
		c.Send(fmt.Sprintf("Send the X-GNOCKER header with this token (%s) to invoke this configuration you can hit the config API /configurations to retrieve all currently configured tokens\n", gnockerToken))
	})

	configApp.Get("/tokens", func(c *fiber.Ctx) {
		var tokens []tokensResponse
		for token, config := range h.tokens {
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
