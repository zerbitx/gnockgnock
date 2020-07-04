package spec

type (
	Configurations map[string]Path

	Path struct {
		TTL   string             `json:"ttl" yaml:"ttl"`
		Paths map[string]Methods `json:"paths" yaml:"paths"`
	}

	Methods map[string]Method

	Method struct {
		ResponseBody         string `json:"responseBody" yaml:"responseBody"`
		ResponseBodyTemplate string `json:"responseBodyTemplate" yaml:"responseBodyTemplate"`
		StatusCode           int    `json:"statusCode" yaml:"statusCode"`
	}
)
