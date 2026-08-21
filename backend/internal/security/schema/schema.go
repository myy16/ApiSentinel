package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Violation struct {
	FieldPath string `json:"fieldPath"`
	Message   string `json:"message"`
	Keyword   string `json:"keyword"`
}

type Validator struct {
	compiled *jsonschema.Schema
}

func NewValidator(schemaJSON string) (*Validator, error) {
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020

	if err := compiler.AddResource("schema.json", strings.NewReader(schemaJSON)); err != nil {
		return nil, fmt.Errorf("invalid JSON schema syntax: %w", err)
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile JSON schema: %w", err)
	}

	return &Validator{compiled: schema}, nil
}

func (v *Validator) Validate(payloadJSON []byte) ([]Violation, error) {
	var val interface{}
	if err := json.Unmarshal(payloadJSON, &val); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}

	err := v.compiled.Validate(val)
	if err == nil {
		return nil, nil
	}

	var violations []Violation
	if validationErr, ok := err.(*jsonschema.ValidationError); ok {
		violations = extractViolations(validationErr)
	} else {
		violations = append(violations, Violation{
			FieldPath: "/",
			Message:   err.Error(),
			Keyword:   "schema",
		})
	}

	return violations, nil
}

func extractViolations(ve *jsonschema.ValidationError) []Violation {
	var list []Violation

	if len(ve.Causes) == 0 {
		field := ve.InstanceLocation
		if field == "" {
			field = "/"
		}
		list = append(list, Violation{
			FieldPath: field,
			Message:   ve.Message,
			Keyword:   ve.KeywordLocation,
		})
		return list
	}

	for _, cause := range ve.Causes {
		list = append(list, extractViolations(cause)...)
	}

	return list
}
