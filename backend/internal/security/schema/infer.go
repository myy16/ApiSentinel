package schema

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"time"

	"github.com/google/uuid"
)

var uuidRegex = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// InferJSONSchema generates a JSON Schema from a sample JSON payload.
func InferJSONSchema(payloadJSON []byte) (string, error) {
	var parsed interface{}
	if err := json.Unmarshal(payloadJSON, &parsed); err != nil {
		return "", fmt.Errorf("invalid JSON payload: %w", err)
	}

	rootSchema := inferValue(parsed)
	if rootMap, ok := rootSchema.(map[string]interface{}); ok {
		rootMap["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	}

	formatted, err := json.MarshalIndent(rootSchema, "", "  ")
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

func inferValue(v interface{}) interface{} {
	if v == nil {
		return map[string]interface{}{"type": "null"}
	}

	switch val := v.(type) {
	case string:
		schema := map[string]interface{}{"type": "string"}
		if format := detectStringFormat(val); format != "" {
			schema["format"] = format
		}
		return schema

	case bool:
		return map[string]interface{}{"type": "boolean"}

	case float64:
		if val == float64(int64(val)) {
			return map[string]interface{}{"type": "integer"}
		}
		return map[string]interface{}{"type": "number"}

	case []interface{}:
		if len(val) == 0 {
			return map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{},
			}
		}
		// Infer from first item (representative)
		itemSchema := inferValue(val[0])
		return map[string]interface{}{
			"type":  "array",
			"items": itemSchema,
		}

	case map[string]interface{}:
		properties := make(map[string]interface{})
		var required []string

		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			properties[k] = inferValue(val[k])
			required = append(required, k)
		}

		schema := map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema

	default:
		return map[string]interface{}{"type": "string"}
	}
}

func detectStringFormat(s string) string {
	if s == "" {
		return ""
	}

	// 1. UUID
	if uuidRegex.MatchString(s) {
		if _, err := uuid.Parse(s); err == nil {
			return "uuid"
		}
	}

	// 2. Date-Time (RFC3339 / ISO8601)
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return "date-time"
	}
	if _, err := time.Parse("2006-01-02T15:04:05.000Z", s); err == nil {
		return "date-time"
	}
	if _, err := time.Parse("2006-01-02", s); err == nil {
		return "date"
	}

	// 3. Email
	if stringsContains(s, "@") && stringsContains(s, ".") {
		if _, err := mail.ParseAddress(s); err == nil {
			return "email"
		}
	}

	// 4. URI / URL
	if stringsHasPrefix(s, "http://") || stringsHasPrefix(s, "https://") {
		if u, err := url.ParseRequestURI(s); err == nil && u.Host != "" {
			return "uri"
		}
	}

	return ""
}

func stringsContains(s, substr string) bool {
	return len(s) >= len(substr) && regexp.MustCompile(regexp.QuoteMeta(substr)).MatchString(s)
}

func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}
