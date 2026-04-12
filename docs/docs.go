// Package docs provides embedded API documentation assets.
package docs

import _ "embed"

// SwaggerSpec contains the raw Swagger/OpenAPI 2.0 YAML specification.
//
//go:embed source-swagger.yaml
var SwaggerSpec []byte
