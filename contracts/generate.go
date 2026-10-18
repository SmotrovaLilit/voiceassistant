// Package contracts owns shared service contracts and their generated transport code.
package contracts

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.6.0 --config voiceassistant-api/oapi-codegen-client.yaml voiceassistant-api/openapi.yaml
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.6.0 --config voiceassistant-api/oapi-codegen-server.yaml voiceassistant-api/openapi.yaml
