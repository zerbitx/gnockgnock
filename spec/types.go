package spec

type (
	Configurations map[string]Path

	Path struct {
		TTL   string             `json:"ttl" yaml:"ttl"`
		Paths map[string]Methods `json:"paths" yaml:"paths"`
	}

	Methods map[string]Method

	Method struct {
		Payload    string `json:"payload" yaml:"payload"`
		StatusCode int    `json:"statusCode" yaml:"statusCode"`
	}
)
