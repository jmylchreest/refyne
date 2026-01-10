package schema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Schema defines the structure for data extraction.
type Schema struct {
	Name        string  `json:"name" yaml:"name"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Fields      []Field `json:"fields" yaml:"fields"`

	target   reflect.Type      // Original struct type for unmarshaling
	validate *validator.Validate
}

// SchemaOption configures schema creation.
type SchemaOption func(*schemaBuilder)

type schemaBuilder struct {
	description string
}

// WithDescription sets the schema description (the NLP context).
func WithDescription(desc string) SchemaOption {
	return func(b *schemaBuilder) {
		b.description = desc
	}
}

// NewSchema creates a Schema from a struct type using reflection.
func NewSchema[T any](opts ...SchemaOption) (Schema, error) {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return Schema{}, fmt.Errorf("schema must be created from a struct type, got %v", t.Kind())
	}

	builder := &schemaBuilder{}
	for _, opt := range opts {
		opt(builder)
	}

	fields, err := extractFields(t)
	if err != nil {
		return Schema{}, err
	}

	return Schema{
		Name:        t.Name(),
		Description: builder.description,
		Fields:      fields,
		target:      t,
		validate:    validator.New(),
	}, nil
}

// FromFile loads a schema from a JSON or YAML file.
func FromFile(path string) (Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Schema{}, fmt.Errorf("failed to read schema file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	var s Schema

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &s); err != nil {
			return Schema{}, fmt.Errorf("failed to parse JSON schema: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &s); err != nil {
			return Schema{}, fmt.Errorf("failed to parse YAML schema: %w", err)
		}
	default:
		return Schema{}, fmt.Errorf("unsupported schema file format: %s", ext)
	}

	s.validate = validator.New()
	return s, nil
}

// FromJSON creates a schema from JSON data.
func FromJSON(data []byte) (Schema, error) {
	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return Schema{}, fmt.Errorf("failed to parse JSON schema: %w", err)
	}
	s.validate = validator.New()
	return s, nil
}

// FromYAML creates a schema from YAML data.
func FromYAML(data []byte) (Schema, error) {
	var s Schema
	if err := yaml.Unmarshal(data, &s); err != nil {
		return Schema{}, fmt.Errorf("failed to parse YAML schema: %w", err)
	}
	s.validate = validator.New()
	return s, nil
}

// extractFields recursively extracts field definitions from a struct type.
func extractFields(t reflect.Type) ([]Field, error) {
	fields := make([]Field, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		field := Field{
			Name:        getJSONName(sf),
			Description: sf.Tag.Get("description"),
			Required:    !hasOmitempty(sf),
			Validators:  parseValidators(sf.Tag.Get("validate")),
		}

		// Handle examples tag
		if examples := sf.Tag.Get("examples"); examples != "" {
			field.Examples = strings.Split(examples, ",")
		}

		// Determine field type
		fieldType := sf.Type
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
			field.Required = false
		}

		switch fieldType.Kind() {
		case reflect.String:
			field.Type = TypeString
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			field.Type = TypeInteger
		case reflect.Float32, reflect.Float64:
			field.Type = TypeNumber
		case reflect.Bool:
			field.Type = TypeBoolean
		case reflect.Slice:
			field.Type = TypeArray
			itemField, err := extractFieldFromType(fieldType.Elem())
			if err != nil {
				return nil, err
			}
			field.Items = &itemField
		case reflect.Struct:
			field.Type = TypeObject
			props, err := extractFields(fieldType)
			if err != nil {
				return nil, err
			}
			field.Properties = props
		case reflect.Map:
			field.Type = TypeObject
		default:
			return nil, fmt.Errorf("unsupported field type: %v for field %s", fieldType.Kind(), sf.Name)
		}

		fields = append(fields, field)
	}

	return fields, nil
}

// extractFieldFromType extracts a Field definition from a reflect.Type.
func extractFieldFromType(t reflect.Type) (Field, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	field := Field{}

	switch t.Kind() {
	case reflect.String:
		field.Type = TypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.Type = TypeInteger
	case reflect.Float32, reflect.Float64:
		field.Type = TypeNumber
	case reflect.Bool:
		field.Type = TypeBoolean
	case reflect.Slice:
		field.Type = TypeArray
		itemField, err := extractFieldFromType(t.Elem())
		if err != nil {
			return Field{}, err
		}
		field.Items = &itemField
	case reflect.Struct:
		field.Type = TypeObject
		props, err := extractFields(t)
		if err != nil {
			return Field{}, err
		}
		field.Properties = props
	default:
		return Field{}, fmt.Errorf("unsupported type: %v", t.Kind())
	}

	return field, nil
}

// getJSONName returns the JSON field name from struct tags.
func getJSONName(sf reflect.StructField) string {
	tag := sf.Tag.Get("json")
	if tag == "" || tag == "-" {
		return sf.Name
	}
	parts := strings.Split(tag, ",")
	if parts[0] != "" {
		return parts[0]
	}
	return sf.Name
}

// hasOmitempty checks if the json tag contains omitempty.
func hasOmitempty(sf reflect.StructField) bool {
	tag := sf.Tag.Get("json")
	return strings.Contains(tag, "omitempty")
}

// parseValidators extracts validator tags.
func parseValidators(tag string) []string {
	if tag == "" {
		return nil
	}
	return strings.Split(tag, ",")
}

// Unmarshal parses JSON into the target struct type.
func (s Schema) Unmarshal(data []byte) (any, error) {
	if s.target == nil {
		// Schema was loaded from file, unmarshal to map
		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal: %w", err)
		}
		return result, nil
	}

	v := reflect.New(s.target).Interface()
	if err := json.Unmarshal(data, v); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	return v, nil
}

// Validate checks the data against validation rules.
func (s Schema) Validate(data any) []ValidationError {
	if s.validate == nil {
		return nil
	}

	// For maps loaded from file schemas, skip struct validation
	if _, ok := data.(map[string]any); ok {
		return s.validateMap(data.(map[string]any))
	}

	// For pointer to struct
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	err := s.validate.Struct(data)
	if err == nil {
		return nil
	}

	var errors []ValidationError
	for _, e := range err.(validator.ValidationErrors) {
		errors = append(errors, ValidationError{
			Field:   e.Field(),
			Message: formatValidationError(e),
			Value:   e.Value(),
		})
	}
	return errors
}

// validateMap validates a map against the schema fields.
func (s Schema) validateMap(data map[string]any) []ValidationError {
	var errors []ValidationError

	for _, field := range s.Fields {
		val, exists := data[field.Name]
		if field.Required && !exists {
			errors = append(errors, ValidationError{
				Field:   field.Name,
				Message: "required field is missing",
			})
			continue
		}
		if !exists {
			continue
		}

		// Type check
		if err := validateFieldType(field, val); err != nil {
			errors = append(errors, ValidationError{
				Field:   field.Name,
				Message: err.Error(),
				Value:   val,
			})
		}
	}

	return errors
}

// validateFieldType checks if a value matches the expected field type.
func validateFieldType(field Field, val any) error {
	if val == nil {
		if field.Required {
			return fmt.Errorf("value is null but field is required")
		}
		return nil
	}

	switch field.Type {
	case TypeString:
		if _, ok := val.(string); !ok {
			return fmt.Errorf("expected string, got %T", val)
		}
	case TypeInteger:
		switch val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float64:
			// float64 is acceptable as JSON numbers are decoded as float64
		default:
			return fmt.Errorf("expected integer, got %T", val)
		}
	case TypeNumber:
		switch val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float64:
			// Accept numeric types
		default:
			return fmt.Errorf("expected number, got %T", val)
		}
	case TypeBoolean:
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", val)
		}
	case TypeArray:
		arr, ok := val.([]any)
		if !ok {
			return fmt.Errorf("expected array, got %T", val)
		}
		if field.Items != nil {
			for i, item := range arr {
				if err := validateFieldType(*field.Items, item); err != nil {
					return fmt.Errorf("item %d: %w", i, err)
				}
			}
		}
	case TypeObject:
		if _, ok := val.(map[string]any); !ok {
			return fmt.Errorf("expected object, got %T", val)
		}
	}

	return nil
}

// formatValidationError creates a human-readable error message.
func formatValidationError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "is required"
	case "min":
		return fmt.Sprintf("must be at least %s", e.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", e.Param())
	case "email":
		return "must be a valid email address"
	case "url":
		return "must be a valid URL"
	default:
		return fmt.Sprintf("failed validation '%s'", e.Tag())
	}
}
