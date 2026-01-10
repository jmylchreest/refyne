// Package schema provides structured data schema definitions for LLM extraction.
package schema

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// FieldType represents the type of a schema field.
type FieldType string

const (
	TypeString  FieldType = "string"
	TypeNumber  FieldType = "number"
	TypeInteger FieldType = "integer"
	TypeBoolean FieldType = "boolean"
	TypeArray   FieldType = "array"
	TypeObject  FieldType = "object"
)

// Field represents a single field in the schema.
type Field struct {
	Name        string    `json:"name,omitempty" yaml:"name,omitempty"`
	Type        FieldType `json:"type" yaml:"type"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool      `json:"required,omitempty" yaml:"required,omitempty"`
	Items       *Field    `json:"items,omitempty" yaml:"items,omitempty"`       // For array types
	Properties  []Field   `json:"-" yaml:"-"`                                   // For object types (populated by custom unmarshal)
	Validators  []string  `json:"validators,omitempty" yaml:"validators,omitempty"` // Validation tags
	Default     any       `json:"default,omitempty" yaml:"default,omitempty"`       // Default value
	Examples    []string  `json:"examples,omitempty" yaml:"examples,omitempty"`     // Example values
}

// fieldAlias is used to avoid infinite recursion in UnmarshalYAML/JSON.
type fieldAlias Field

// fieldRaw is used for initial unmarshaling to detect properties format.
type fieldRaw struct {
	fieldAlias `yaml:",inline"`
	// Properties can be either a map (YAML-style) or array (JSON-style)
	PropertiesRaw yaml.Node `yaml:"properties"`
}

// UnmarshalYAML implements custom YAML unmarshaling to handle both map and array properties.
func (f *Field) UnmarshalYAML(node *yaml.Node) error {
	var raw fieldRaw
	if err := node.Decode(&raw); err != nil {
		return err
	}

	*f = Field(raw.fieldAlias)

	// Handle properties if present
	if raw.PropertiesRaw.Kind != 0 {
		switch raw.PropertiesRaw.Kind {
		case yaml.MappingNode:
			// Map format: properties: {name: {type: string}, ...}
			var propsMap map[string]Field
			if err := raw.PropertiesRaw.Decode(&propsMap); err != nil {
				return err
			}
			for name, prop := range propsMap {
				prop.Name = name
				f.Properties = append(f.Properties, prop)
			}
		case yaml.SequenceNode:
			// Array format: properties: [{name: x, type: string}, ...]
			if err := raw.PropertiesRaw.Decode(&f.Properties); err != nil {
				return err
			}
		}
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling to include properties.
func (f Field) MarshalJSON() ([]byte, error) {
	type fieldJSON struct {
		Name        string    `json:"name,omitempty"`
		Type        FieldType `json:"type"`
		Description string    `json:"description,omitempty"`
		Required    bool      `json:"required,omitempty"`
		Items       *Field    `json:"items,omitempty"`
		Properties  []Field   `json:"properties,omitempty"`
		Validators  []string  `json:"validators,omitempty"`
		Default     any       `json:"default,omitempty"`
		Examples    []string  `json:"examples,omitempty"`
	}

	//nolint:staticcheck // S1016: Can't use conversion - Field.Properties has json:"-" tag
	return json.Marshal(fieldJSON{
		Name:        f.Name,
		Type:        f.Type,
		Description: f.Description,
		Required:    f.Required,
		Items:       f.Items,
		Properties:  f.Properties,
		Validators:  f.Validators,
		Default:     f.Default,
		Examples:    f.Examples,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling to handle both map and array properties.
func (f *Field) UnmarshalJSON(data []byte) error {
	type fieldJSON struct {
		Name        string          `json:"name,omitempty"`
		Type        FieldType       `json:"type"`
		Description string          `json:"description,omitempty"`
		Required    bool            `json:"required,omitempty"`
		Items       *Field          `json:"items,omitempty"`
		Properties  json.RawMessage `json:"properties,omitempty"`
		Validators  []string        `json:"validators,omitempty"`
		Default     any             `json:"default,omitempty"`
		Examples    []string        `json:"examples,omitempty"`
	}

	var raw fieldJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	f.Name = raw.Name
	f.Type = raw.Type
	f.Description = raw.Description
	f.Required = raw.Required
	f.Items = raw.Items
	f.Validators = raw.Validators
	f.Default = raw.Default
	f.Examples = raw.Examples

	// Handle properties if present
	if len(raw.Properties) > 0 {
		// Try array format first
		var propsArray []Field
		if err := json.Unmarshal(raw.Properties, &propsArray); err == nil {
			f.Properties = propsArray
		} else {
			// Try map format
			var propsMap map[string]Field
			if err := json.Unmarshal(raw.Properties, &propsMap); err != nil {
				return err
			}
			for name, prop := range propsMap {
				prop.Name = name
				f.Properties = append(f.Properties, prop)
			}
		}
	}

	return nil
}

// ValidationError represents a validation failure.
type ValidationError struct {
	Field   string
	Message string
	Value   any
}

func (e ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
