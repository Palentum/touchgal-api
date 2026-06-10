package openapi

import _ "embed"

// Document returns the static OpenAPI contract served by the backend docs endpoint.
func Document() string {
	return document
}

//go:embed openapi.yaml
var document string
