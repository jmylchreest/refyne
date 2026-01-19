package schema

import (
	"strings"
)

// ToJSONSchema converts the schema to JSON Schema format for LLM structured output.
func (s Schema) ToJSONSchema() (map[string]any, error) {
	properties := make(map[string]any)
	required := make([]string, 0)

	for _, field := range s.Fields {
		properties[field.Name] = fieldToJSONSchema(field)
		if field.Required {
			required = append(required, field.Name)
		}
	}

	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false, // Required for strict mode (OpenRouter/OpenAI)
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	if s.Description != "" {
		schema["description"] = s.Description
	}

	return schema, nil
}

// fieldToJSONSchema converts a Field to JSON Schema format.
func fieldToJSONSchema(f Field) map[string]any {
	schema := map[string]any{
		"type": string(f.Type),
	}

	if f.Description != "" {
		schema["description"] = f.Description
	}

	if len(f.Examples) > 0 {
		schema["examples"] = f.Examples
	}

	if f.Default != nil {
		schema["default"] = f.Default
	}

	// Handle array items
	if f.Type == TypeArray && f.Items != nil {
		schema["items"] = fieldToJSONSchema(*f.Items)
	}

	// Handle object properties
	if f.Type == TypeObject && len(f.Properties) > 0 {
		props := make(map[string]any)
		req := make([]string, 0)
		for _, p := range f.Properties {
			props[p.Name] = fieldToJSONSchema(p)
			if p.Required {
				req = append(req, p.Name)
			}
		}
		schema["properties"] = props
		schema["additionalProperties"] = false // Required for strict mode
		if len(req) > 0 {
			schema["required"] = req
		}
	}

	return schema
}

// ToPromptDescription generates a human-readable description for the LLM prompt.
func (s Schema) ToPromptDescription() string {
	var sb strings.Builder

	sb.WriteString("## Content Type\n")
	if s.Description != "" {
		sb.WriteString(s.Description)
	} else {
		sb.WriteString("Extract the following structured data.\n")
	}
	sb.WriteString("\n\n## Fields to Extract\n")

	for _, field := range s.Fields {
		writeFieldDescription(&sb, field, 0)
	}

	return sb.String()
}

// writeFieldDescription writes a field description to the string builder.
func writeFieldDescription(sb *strings.Builder, f Field, indent int) {
	prefix := strings.Repeat("  ", indent)

	// Field name and type
	sb.WriteString(prefix)
	sb.WriteString("- ")
	sb.WriteString(f.Name)
	sb.WriteString(" (")
	sb.WriteString(string(f.Type))

	if f.Required {
		sb.WriteString(", required")
	}

	sb.WriteString(")")

	// Description
	if f.Description != "" {
		sb.WriteString(": ")
		sb.WriteString(f.Description)
	}

	sb.WriteString("\n")

	// Array items description
	if f.Type == TypeArray && f.Items != nil && f.Items.Type == TypeObject {
		sb.WriteString(prefix)
		sb.WriteString("  Each item:\n")
		for _, prop := range f.Items.Properties {
			writeFieldDescription(sb, prop, indent+2)
		}
	}

	// Nested object properties
	if f.Type == TypeObject && len(f.Properties) > 0 {
		for _, prop := range f.Properties {
			writeFieldDescription(sb, prop, indent+1)
		}
	}
}
