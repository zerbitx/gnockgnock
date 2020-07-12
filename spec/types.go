package spec

type (
	// Configurations is a mapping from name to a set of path/method expectations
	Configurations map[string]Configuration

	// Configuration holds the TTL for the config to be gnockable (no TTL means live forever)
	// Paths hold each path configuration
	Configuration struct {
		TTL   string               `json:"ttl" yaml:"ttl"`
		Paths map[string]Responses `json:"paths" yaml:"paths"`
	}

	// Responses map each method's response for a given path
	Responses map[string]Response

	// Response configures how gnock should response.
	Response struct {
		Body         string              `json:"body" yaml:"body"`
		BodyTemplate string              `json:"bodyTemplate" yaml:"bodyTemplate"`
		StatusCode   int                 `json:"statusCode" yaml:"statusCode"`
		Headers      []map[string]string `json:"responseHeaders" yaml:"responseHeaders"`
	}
)
