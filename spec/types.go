package spec

type (
	// Configurations is a mapping from name to a set of path/method expectations
	Configurations map[string]Configuration

	// Configuration holds the TTL for the config to be gnockable (no TTL means live forever)
	// Paths hold each path configuration
	Configuration struct {
		TTL   string             `json:"ttl" yaml:"ttl"`
		Paths map[string]Methods `json:"paths" yaml:"paths"`
	}

	// Methods map each method's response for a given path
	Methods map[string]Method

	// Method configures how gnock should response.
	Method struct {
		ResponseBody         string `json:"responseBody" yaml:"responseBody"`
		ResponseBodyTemplate string `json:"responseBodyTemplate" yaml:"responseBodyTemplate"`
		StatusCode           int    `json:"statusCode" yaml:"statusCode"`
	}
)
