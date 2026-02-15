package api

import _ "embed"

// OpenAPISpec — OpenAPI 2.0 (Swagger) спецификация для Operator Directory API.
// Обновить после смены контракта: make proto-openapi или отредактировать api/openapi.json.
//
//go:embed openapi.json
var OpenAPISpec []byte
