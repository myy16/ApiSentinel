package schema

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// DriftChangeType represents the kind of schema change detected.
type DriftChangeType string

const (
	ChangeFieldAdded    DriftChangeType = "FIELD_ADDED"
	ChangeFieldMissing  DriftChangeType = "FIELD_MISSING"
	ChangeTypeMismatch  DriftChangeType = "TYPE_MISMATCH"
)

// DriftChange details a specific discrepancy between baseline schema and actual payload.
type DriftChange struct {
	Path        string          `json:"path"`
	ChangeType  DriftChangeType `json:"changeType"`
	Expected    string          `json:"expected,omitempty"`
	Actual      string          `json:"actual,omitempty"`
	Description string          `json:"description"`
}

// DriftReport encapsulates the full drift analysis result.
type DriftReport struct {
	HasDrift bool          `json:"hasDrift"`
	Severity string        `json:"severity"` // "BREAKING", "NON_BREAKING", "NONE"
	Changes  []DriftChange `json:"changes"`
	Summary  string        `json:"summary"`
}

// DetectDrift compares a JSON payload against a baseline JSON Schema and produces a structural diff report.
func DetectDrift(baselineSchemaJSON []byte, payloadJSON []byte) (DriftReport, error) {
	var schemaMap map[string]interface{}
	if err := json.Unmarshal(baselineSchemaJSON, &schemaMap); err != nil {
		return DriftReport{}, fmt.Errorf("invalid baseline schema JSON: %w", err)
	}

	var payloadVal interface{}
	if err := json.Unmarshal(payloadJSON, &payloadVal); err != nil {
		return DriftReport{}, fmt.Errorf("invalid payload JSON: %w", err)
	}

	var changes []DriftChange
	compareNode("", schemaMap, payloadVal, &changes)

	if len(changes) == 0 {
		return DriftReport{
			HasDrift: false,
			Severity: "NONE",
			Changes:  nil,
			Summary:  "Payload aktif baseline şemasıyla tam uyumlu.",
		}, nil
	}

	severity := "NON_BREAKING"
	for _, c := range changes {
		if c.ChangeType == ChangeFieldMissing || c.ChangeType == ChangeTypeMismatch {
			severity = "BREAKING"
			break
		}
	}

	summary := fmt.Sprintf("%d adet şema sapması tespit edildi (Önem: %s).", len(changes), severity)

	return DriftReport{
		HasDrift: true,
		Severity: severity,
		Changes:  changes,
		Summary:  summary,
	}, nil
}

func compareNode(currentPath string, schemaNode map[string]interface{}, payloadNode interface{}, changes *[]DriftChange) {
	if schemaNode == nil {
		return
	}

	expectedType, _ := schemaNode["type"].(string)

	// 1. Type validation
	if expectedType != "" && payloadNode != nil {
		actualType := getJSONType(payloadNode)
		if !isTypeCompatible(expectedType, actualType) {
			path := currentPath
			if path == "" {
				path = "$"
			}
			*changes = append(*changes, DriftChange{
				Path:        path,
				ChangeType:  ChangeTypeMismatch,
				Expected:    expectedType,
				Actual:      actualType,
				Description: fmt.Sprintf("'%s' alanında beklenen tip '%s', ancak '%s' geldi.", path, expectedType, actualType),
			})
			return
		}
	}

	// 2. Object property comparison
	if expectedType == "object" || expectedType == "" {
		payloadMap, isMap := payloadNode.(map[string]interface{})
		if !isMap && payloadNode != nil {
			return
		}

		schemaProps, _ := schemaNode["properties"].(map[string]interface{})
		if schemaProps == nil {
			schemaProps = make(map[string]interface{})
		}

		// Check required fields
		if reqList, ok := schemaNode["required"].([]interface{}); ok {
			for _, reqItem := range reqList {
				reqFieldName, ok := reqItem.(string)
				if !ok {
					continue
				}
				if payloadMap == nil || payloadMap[reqFieldName] == nil {
					fieldPath := appendPath(currentPath, reqFieldName)
					*changes = append(*changes, DriftChange{
						Path:        fieldPath,
						ChangeType:  ChangeFieldMissing,
						Expected:    "required",
						Actual:      "missing",
						Description: fmt.Sprintf("Zorunlu alan '%s' payload içinde bulunamadı.", fieldPath),
					})
				}
			}
		}

		if payloadMap == nil {
			return
		}

		// Check for added fields (present in payload but absent in schema properties)
		payloadKeys := make([]string, 0, len(payloadMap))
		for k := range payloadMap {
			payloadKeys = append(payloadKeys, k)
		}
		sort.Strings(payloadKeys)

		for _, k := range payloadKeys {
			propVal := payloadMap[k]
			subSchema, exists := schemaProps[k].(map[string]interface{})
			fieldPath := appendPath(currentPath, k)

			if !exists {
				// New field added!
				*changes = append(*changes, DriftChange{
					Path:        fieldPath,
					ChangeType:  ChangeFieldAdded,
					Expected:    "undefined",
					Actual:      getJSONType(propVal),
					Description: fmt.Sprintf("Yeni alan '%s' (%s) payload içinde tespit edildi.", fieldPath, getJSONType(propVal)),
				})
			} else {
				// Recursively compare child
				compareNode(fieldPath, subSchema, propVal, changes)
			}
		}
	}

	// 3. Array items comparison
	if expectedType == "array" {
		payloadArr, isArr := payloadNode.([]interface{})
		if !isArr || len(payloadArr) == 0 {
			return
		}

		itemsSchema, _ := schemaNode["items"].(map[string]interface{})
		if itemsSchema != nil && len(payloadArr) > 0 {
			// Compare first element as representative
			compareNode(appendPath(currentPath, "[*]"), itemsSchema, payloadArr[0], changes)
		}
	}
}

func appendPath(base, child string) string {
	if base == "" {
		if strings.HasPrefix(child, "[") {
			return "$" + child
		}
		return "$." + child
	}
	if strings.HasPrefix(child, "[") {
		return base + child
	}
	return base + "." + child
}

func getJSONType(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		if val == float64(int64(val)) {
			return "integer"
		}
		return "number"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "string"
	}
}

func isTypeCompatible(expected, actual string) bool {
	if expected == actual {
		return true
	}
	if expected == "number" && actual == "integer" {
		return true
	}
	return false
}
