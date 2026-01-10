// Package schema provides structured data schema definitions for LLM extraction.
package schema

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
	Name        string    `json:"name" yaml:"name"`
	Type        FieldType `json:"type" yaml:"type"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool      `json:"required,omitempty" yaml:"required,omitempty"`
	Items       *Field    `json:"items,omitempty" yaml:"items,omitempty"`           // For array types
	Properties  []Field   `json:"properties,omitempty" yaml:"properties,omitempty"` // For object types
	Validators  []string  `json:"validators,omitempty" yaml:"validators,omitempty"` // Validation tags
	Default     any       `json:"default,omitempty" yaml:"default,omitempty"`       // Default value
	Examples    []string  `json:"examples,omitempty" yaml:"examples,omitempty"`     // Example values
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
