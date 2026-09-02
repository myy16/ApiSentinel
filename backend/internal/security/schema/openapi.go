package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ExtractSchemaFromOpenAPI parses an OpenAPI 3.0 / 3.1 spec (JSON or YAML) and extracts the webhook request schema.
func ExtractSchemaFromOpenAPI(specBytes []byte, operationPath string) (string, error) {
	var parsed map[string]interface{}

	// Try parsing as JSON first, fallback to YAML
	if err := json.Unmarshal(specBytes, &parsed); err != nil {
		if yamlErr := yaml.Unmarshal(specBytes, &parsed); yamlErr != nil {
			return "", fmt.Errorf("spec is neither valid JSON nor YAML: %w", err)
		}
	}

	// 1. Check OpenAPI version
	openapiVer, _ := parsed["openapi"].(string)
	swaggerVer, _ := parsed["swagger"].(string)
	if openapiVer == "" && swaggerVer == "" {
		return "", fmt.Errorf("missing 'openapi' or 'swagger' root property")
	}

	// 2. Search in "webhooks" (OpenAPI 3.1)
	if webhooks, ok := parsed["webhooks"].(map[string]interface{}); ok && len(webhooks) > 0 {
		for _, whObj := range webhooks {
			if schemaStr, err := extractFromPathItem(whObj, parsed); err == nil && schemaStr != "" {
				return schemaStr, nil
			}
		}
	}

	// 3. Search in "paths" (OpenAPI 3.0 / 3.1)
	if paths, ok := parsed["paths"].(map[string]interface{}); ok && len(paths) > 0 {
		if operationPath != "" {
			if pathItem, ok := paths[operationPath]; ok {
				if schemaStr, err := extractFromPathItem(pathItem, parsed); err == nil && schemaStr != "" {
					return schemaStr, nil
				}
			}
		}
		// Otherwise extract from first available POST/PUT path
		for _, pathItem := range paths {
			if schemaStr, err := extractFromPathItem(pathItem, parsed); err == nil && schemaStr != "" {
				return schemaStr, nil
			}
		}
	}

	// 4. Fallback: Search in "components.schemas" (first schema definition)
	if components, ok := parsed["components"].(map[string]interface{}); ok {
		if schemas, ok := components["schemas"].(map[string]interface{}); ok && len(schemas) > 0 {
			for _, schemaObj := range schemas {
				if schemaMap, ok := schemaObj.(map[string]interface{}); ok {
					schemaMap["$schema"] = "https://json-schema.org/draft/2020-12/schema"
					formatted, err := json.MarshalIndent(schemaMap, "", "  ")
					if err == nil {
						return string(formatted), nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no webhook requestBody schema found in OpenAPI spec")
}

func extractFromPathItem(pathItem interface{}, rootDoc map[string]interface{}) (string, error) {
	pathMap, ok := pathItem.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid pathItem")
	}

	// Check methods: post, put, patch
	for _, method := range []string{"post", "put", "patch"} {
		opObj, ok := pathMap[method]
		if !ok {
			continue
		}
		opMap, ok := opObj.(map[string]interface{})
		if !ok {
			continue
		}

		reqBody, ok := opMap["requestBody"].(map[string]interface{})
		if !ok {
			continue
		}

		// Resolve $ref if requestBody is referenced
		if ref, ok := reqBody["$ref"].(string); ok {
			resolved := resolveRef(ref, rootDoc)
			if rMap, ok := resolved.(map[string]interface{}); ok {
				reqBody = rMap
			}
		}

		content, ok := reqBody["content"].(map[string]interface{})
		if !ok {
			continue
		}

		// Look for application/json or */*
		for _, mediaType := range []string{"application/json", "*/*"} {
			mediaObj, ok := content[mediaType].(map[string]interface{})
			if !ok {
				continue
			}
			schemaObj, ok := mediaObj["schema"]
			if !ok {
				continue
			}

			// If schema has $ref, resolve it
			if schemaMap, ok := schemaObj.(map[string]interface{}); ok {
				if ref, ok := schemaMap["$ref"].(string); ok {
					resolved := resolveRef(ref, rootDoc)
					if rMap, ok := resolved.(map[string]interface{}); ok {
						schemaMap = rMap
					}
				}
				schemaMap["$schema"] = "https://json-schema.org/draft/2020-12/schema"
				formatted, err := json.MarshalIndent(schemaMap, "", "  ")
				if err == nil {
					return string(formatted), nil
				}
			}
		}
	}

	return "", fmt.Errorf("no matching request body schema found")
}

func resolveRef(ref string, rootDoc map[string]interface{}) interface{} {
	if !strings.HasPrefix(ref, "#/") {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
	var current interface{} = rootDoc

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}
