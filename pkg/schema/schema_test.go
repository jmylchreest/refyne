package schema

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// Test structs for NewSchema

type SimpleStruct struct {
	Name  string `json:"name" description:"The name"`
	Age   int    `json:"age" description:"The age in years"`
	Email string `json:"email,omitempty" description:"Optional email"`
}

type NestedStruct struct {
	Title   string `json:"title" description:"Title"`
	Address struct {
		Street string `json:"street" description:"Street address"`
		City   string `json:"city" description:"City name"`
	} `json:"address" description:"Address details"`
}

type StructWithPointer struct {
	Name     string  `json:"name" description:"Required name"`
	Nickname *string `json:"nickname,omitempty" description:"Optional nickname"`
}

type StructWithSlice struct {
	Title       string   `json:"title" description:"Title"`
	Tags        []string `json:"tags" description:"List of tags"`
	Ingredients []struct {
		Name   string  `json:"name" description:"Ingredient name"`
		Amount float64 `json:"amount" description:"Amount needed"`
	} `json:"ingredients" description:"Recipe ingredients"`
}

type StructWithAllTypes struct {
	StringField  string  `json:"string_field"`
	IntField     int     `json:"int_field"`
	Int64Field   int64   `json:"int64_field"`
	Float32Field float32 `json:"float32_field"`
	Float64Field float64 `json:"float64_field"`
	BoolField    bool    `json:"bool_field"`
}

type StructWithValidators struct {
	Email string `json:"email" validate:"required,email"`
	URL   string `json:"url" validate:"url"`
	Name  string `json:"name" validate:"min=2,max=100"`
}

type StructWithExamples struct {
	Status string `json:"status" examples:"active,pending,completed"`
}

// TestNewSchema_BasicStruct tests schema creation from a simple struct
func TestNewSchema_BasicStruct(t *testing.T) {
	s, err := NewSchema[SimpleStruct]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	if s.Name != "SimpleStruct" {
		t.Errorf("expected Name 'SimpleStruct', got %q", s.Name)
	}

	if len(s.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(s.Fields))
	}

	// Check field details
	fieldMap := make(map[string]Field)
	for _, f := range s.Fields {
		fieldMap[f.Name] = f
	}

	nameField := fieldMap["name"]
	if nameField.Type != TypeString {
		t.Errorf("expected name field type 'string', got %q", nameField.Type)
	}
	if !nameField.Required {
		t.Error("expected name field to be required")
	}
	if nameField.Description != "The name" {
		t.Errorf("expected description 'The name', got %q", nameField.Description)
	}

	ageField := fieldMap["age"]
	if ageField.Type != TypeInteger {
		t.Errorf("expected age field type 'integer', got %q", ageField.Type)
	}

	emailField := fieldMap["email"]
	if emailField.Required {
		t.Error("expected email field to be optional (has omitempty)")
	}
}

// TestNewSchema_NestedStruct tests schema creation with nested objects
func TestNewSchema_NestedStruct(t *testing.T) {
	s, err := NewSchema[NestedStruct]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	if len(s.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(s.Fields))
	}

	// Find address field
	var addressField Field
	for _, f := range s.Fields {
		if f.Name == "address" {
			addressField = f
			break
		}
	}

	if addressField.Type != TypeObject {
		t.Errorf("expected address field type 'object', got %q", addressField.Type)
	}

	if len(addressField.Properties) != 2 {
		t.Errorf("expected 2 properties in address, got %d", len(addressField.Properties))
	}
}

// TestNewSchema_WithPointerFields tests that pointer fields are marked optional
func TestNewSchema_WithPointerFields(t *testing.T) {
	s, err := NewSchema[StructWithPointer]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	fieldMap := make(map[string]Field)
	for _, f := range s.Fields {
		fieldMap[f.Name] = f
	}

	if !fieldMap["name"].Required {
		t.Error("expected name field to be required")
	}

	if fieldMap["nickname"].Required {
		t.Error("expected nickname field to be optional (is pointer)")
	}
}

// TestNewSchema_WithSliceFields tests array type detection
func TestNewSchema_WithSliceFields(t *testing.T) {
	s, err := NewSchema[StructWithSlice]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	fieldMap := make(map[string]Field)
	for _, f := range s.Fields {
		fieldMap[f.Name] = f
	}

	tagsField := fieldMap["tags"]
	if tagsField.Type != TypeArray {
		t.Errorf("expected tags field type 'array', got %q", tagsField.Type)
	}
	if tagsField.Items == nil {
		t.Fatal("expected tags.Items to be set")
	}
	if tagsField.Items.Type != TypeString {
		t.Errorf("expected tags items type 'string', got %q", tagsField.Items.Type)
	}

	ingredientsField := fieldMap["ingredients"]
	if ingredientsField.Type != TypeArray {
		t.Errorf("expected ingredients field type 'array', got %q", ingredientsField.Type)
	}
	if ingredientsField.Items == nil {
		t.Fatal("expected ingredients.Items to be set")
	}
	if ingredientsField.Items.Type != TypeObject {
		t.Errorf("expected ingredients items type 'object', got %q", ingredientsField.Items.Type)
	}
	if len(ingredientsField.Items.Properties) != 2 {
		t.Errorf("expected 2 properties in ingredient items, got %d", len(ingredientsField.Items.Properties))
	}
}

// TestNewSchema_NonStructType_Error tests error on non-struct types
func TestNewSchema_NonStructType_Error(t *testing.T) {
	_, err := NewSchema[string]()
	if err == nil {
		t.Fatal("expected error for non-struct type")
	}
	if !strings.Contains(err.Error(), "struct type") {
		t.Errorf("expected error about struct type, got: %v", err)
	}
}

// TestNewSchema_WithDescription tests the WithDescription option
func TestNewSchema_WithDescription(t *testing.T) {
	desc := "A simple test schema for demonstration"
	s, err := NewSchema[SimpleStruct](WithDescription(desc))
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	if s.Description != desc {
		t.Errorf("expected description %q, got %q", desc, s.Description)
	}
}

// TestNewSchema_AllNumericTypes tests all numeric type mappings
func TestNewSchema_AllNumericTypes(t *testing.T) {
	s, err := NewSchema[StructWithAllTypes]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	fieldMap := make(map[string]Field)
	for _, f := range s.Fields {
		fieldMap[f.Name] = f
	}

	tests := []struct {
		name     string
		expected FieldType
	}{
		{"string_field", TypeString},
		{"int_field", TypeInteger},
		{"int64_field", TypeInteger},
		{"float32_field", TypeNumber},
		{"float64_field", TypeNumber},
		{"bool_field", TypeBoolean},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := fieldMap[tt.name]
			if field.Type != tt.expected {
				t.Errorf("expected %s type %q, got %q", tt.name, tt.expected, field.Type)
			}
		})
	}
}

// TestNewSchema_WithValidators tests validator tag parsing
func TestNewSchema_WithValidators(t *testing.T) {
	s, err := NewSchema[StructWithValidators]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	fieldMap := make(map[string]Field)
	for _, f := range s.Fields {
		fieldMap[f.Name] = f
	}

	emailValidators := fieldMap["email"].Validators
	if len(emailValidators) != 2 {
		t.Errorf("expected 2 validators for email, got %d", len(emailValidators))
	}
}

// TestNewSchema_WithExamples tests examples tag parsing
func TestNewSchema_WithExamples(t *testing.T) {
	s, err := NewSchema[StructWithExamples]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	var statusField Field
	for _, f := range s.Fields {
		if f.Name == "status" {
			statusField = f
			break
		}
	}

	if len(statusField.Examples) != 3 {
		t.Errorf("expected 3 examples, got %d", len(statusField.Examples))
	}

	expected := []string{"active", "pending", "completed"}
	if !reflect.DeepEqual(statusField.Examples, expected) {
		t.Errorf("expected examples %v, got %v", expected, statusField.Examples)
	}
}

// TestFromJSON_ValidSchema tests JSON schema parsing
func TestFromJSON_ValidSchema(t *testing.T) {
	jsonData := []byte(`{
		"name": "TestSchema",
		"description": "A test schema",
		"fields": [
			{"name": "title", "type": "string", "required": true},
			{"name": "count", "type": "integer", "required": false}
		]
	}`)

	s, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if s.Name != "TestSchema" {
		t.Errorf("expected name 'TestSchema', got %q", s.Name)
	}

	if s.Description != "A test schema" {
		t.Errorf("expected description 'A test schema', got %q", s.Description)
	}

	if len(s.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(s.Fields))
	}

	if s.Fields[0].Name != "title" {
		t.Errorf("expected first field name 'title', got %q", s.Fields[0].Name)
	}

	if !s.Fields[0].Required {
		t.Error("expected title field to be required")
	}
}

// TestFromJSON_InvalidJSON tests error handling for invalid JSON
func TestFromJSON_InvalidJSON(t *testing.T) {
	jsonData := []byte(`{invalid json}`)

	_, err := FromJSON(jsonData)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestFromJSON_NestedProperties tests JSON with nested object properties
func TestFromJSON_NestedProperties(t *testing.T) {
	jsonData := []byte(`{
		"name": "NestedSchema",
		"fields": [
			{
				"name": "address",
				"type": "object",
				"properties": [
					{"name": "street", "type": "string"},
					{"name": "city", "type": "string"}
				]
			}
		]
	}`)

	s, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if len(s.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(s.Fields))
	}

	addressField := s.Fields[0]
	if addressField.Type != TypeObject {
		t.Errorf("expected address type 'object', got %q", addressField.Type)
	}

	if len(addressField.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(addressField.Properties))
	}
}

// TestFromYAML_ValidSchema tests YAML schema parsing
func TestFromYAML_ValidSchema(t *testing.T) {
	yamlData := []byte(`
name: TestSchema
description: A test schema from YAML
fields:
  - name: title
    type: string
    required: true
  - name: rating
    type: number
`)

	s, err := FromYAML(yamlData)
	if err != nil {
		t.Fatalf("FromYAML failed: %v", err)
	}

	if s.Name != "TestSchema" {
		t.Errorf("expected name 'TestSchema', got %q", s.Name)
	}

	if len(s.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(s.Fields))
	}

	if s.Fields[0].Type != TypeString {
		t.Errorf("expected title type 'string', got %q", s.Fields[0].Type)
	}

	if s.Fields[1].Type != TypeNumber {
		t.Errorf("expected rating type 'number', got %q", s.Fields[1].Type)
	}
}

// TestFromYAML_InvalidYAML tests error handling for invalid YAML
func TestFromYAML_InvalidYAML(t *testing.T) {
	yamlData := []byte(`
name: TestSchema
fields:
  - invalid: [unclosed
`)

	_, err := FromYAML(yamlData)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// TestFromYAML_MapProperties tests YAML with map-style properties
func TestFromYAML_MapProperties(t *testing.T) {
	yamlData := []byte(`
name: MapPropsSchema
fields:
  - name: person
    type: object
    properties:
      name:
        type: string
        required: true
      age:
        type: integer
`)

	s, err := FromYAML(yamlData)
	if err != nil {
		t.Fatalf("FromYAML failed: %v", err)
	}

	personField := s.Fields[0]
	if len(personField.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(personField.Properties))
	}

	// Check that property names are populated from map keys
	propNames := make(map[string]bool)
	for _, p := range personField.Properties {
		propNames[p.Name] = true
	}

	if !propNames["name"] || !propNames["age"] {
		t.Errorf("expected properties 'name' and 'age', got %v", propNames)
	}
}

// TestValidate_StructData_Valid tests validation of valid struct data
func TestValidate_StructData_Valid(t *testing.T) {
	type ValidateStruct struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	s, err := NewSchema[ValidateStruct]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	data := &ValidateStruct{
		Name:  "John",
		Email: "john@example.com",
	}

	errors := s.Validate(data)
	if len(errors) != 0 {
		t.Errorf("expected no validation errors, got %v", errors)
	}
}

// TestValidate_StructData_Invalid tests validation of invalid struct data
func TestValidate_StructData_Invalid(t *testing.T) {
	type ValidateStruct struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	s, err := NewSchema[ValidateStruct]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	data := &ValidateStruct{
		Name:  "",
		Email: "invalid-email",
	}

	errors := s.Validate(data)
	if len(errors) != 2 {
		t.Errorf("expected 2 validation errors, got %d: %v", len(errors), errors)
	}
}

// TestValidateMap_RequiredField tests map validation for required fields
func TestValidateMap_RequiredField(t *testing.T) {
	jsonData := []byte(`{
		"name": "TestSchema",
		"fields": [
			{"name": "title", "type": "string", "required": true},
			{"name": "optional", "type": "string", "required": false}
		]
	}`)

	s, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	tests := []struct {
		name      string
		data      map[string]any
		wantErrs  int
		errSubstr string
	}{
		{
			name:     "valid_with_required",
			data:     map[string]any{"title": "Hello"},
			wantErrs: 0,
		},
		{
			name:      "missing_required",
			data:      map[string]any{"optional": "value"},
			wantErrs:  1,
			errSubstr: "required",
		},
		{
			name:     "all_fields_present",
			data:     map[string]any{"title": "Hello", "optional": "World"},
			wantErrs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := s.Validate(tt.data)
			if len(errors) != tt.wantErrs {
				t.Errorf("expected %d errors, got %d: %v", tt.wantErrs, len(errors), errors)
			}
			if tt.errSubstr != "" && len(errors) > 0 {
				if !strings.Contains(errors[0].Message, tt.errSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errSubstr, errors[0].Message)
				}
			}
		})
	}
}

// TestValidateFieldType tests type validation for different field types
func TestValidateFieldType(t *testing.T) {
	tests := []struct {
		name    string
		field   Field
		value   any
		wantErr bool
	}{
		// String tests
		{"string_valid", Field{Type: TypeString}, "hello", false},
		{"string_invalid", Field{Type: TypeString}, 123, true},

		// Integer tests
		{"integer_from_int", Field{Type: TypeInteger}, 42, false},
		{"integer_from_float64", Field{Type: TypeInteger}, float64(42), false},
		{"integer_invalid", Field{Type: TypeInteger}, "not a number", true},

		// Number tests
		{"number_from_int", Field{Type: TypeNumber}, 42, false},
		{"number_from_float64", Field{Type: TypeNumber}, 3.14, false},
		{"number_invalid", Field{Type: TypeNumber}, "not a number", true},

		// Boolean tests
		{"boolean_valid", Field{Type: TypeBoolean}, true, false},
		{"boolean_invalid", Field{Type: TypeBoolean}, "true", true},

		// Array tests
		{"array_valid", Field{Type: TypeArray}, []any{"a", "b"}, false},
		{"array_invalid", Field{Type: TypeArray}, "not an array", true},
		{
			"array_items_valid",
			Field{Type: TypeArray, Items: &Field{Type: TypeString}},
			[]any{"a", "b"},
			false,
		},
		{
			"array_items_invalid",
			Field{Type: TypeArray, Items: &Field{Type: TypeString}},
			[]any{"a", 123},
			true,
		},

		// Object tests
		{"object_valid", Field{Type: TypeObject}, map[string]any{"key": "value"}, false},
		{"object_invalid", Field{Type: TypeObject}, "not an object", true},

		// Null handling
		{"null_optional", Field{Type: TypeString, Required: false}, nil, false},
		{"null_required", Field{Type: TypeString, Required: true}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFieldType(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFieldType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestUnmarshal_WithTargetType tests unmarshaling to the original struct type
func TestUnmarshal_WithTargetType(t *testing.T) {
	s, err := NewSchema[SimpleStruct]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	jsonData := []byte(`{"name": "John", "age": 30, "email": "john@example.com"}`)

	result, err := s.Unmarshal(jsonData)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	ss, ok := result.(*SimpleStruct)
	if !ok {
		t.Fatalf("expected *SimpleStruct, got %T", result)
	}

	if ss.Name != "John" {
		t.Errorf("expected Name 'John', got %q", ss.Name)
	}

	if ss.Age != 30 {
		t.Errorf("expected Age 30, got %d", ss.Age)
	}
}

// TestUnmarshal_NoTargetType tests unmarshaling to map when loaded from file
func TestUnmarshal_NoTargetType(t *testing.T) {
	jsonSchema := []byte(`{
		"name": "FileSchema",
		"fields": [{"name": "title", "type": "string"}]
	}`)

	s, err := FromJSON(jsonSchema)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	jsonData := []byte(`{"title": "Hello World", "extra": "data"}`)

	result, err := s.Unmarshal(jsonData)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result)
	}

	if m["title"] != "Hello World" {
		t.Errorf("expected title 'Hello World', got %v", m["title"])
	}
}

// TestUnmarshal_InvalidJSON tests error handling for invalid JSON
func TestUnmarshal_InvalidJSON(t *testing.T) {
	s, _ := NewSchema[SimpleStruct]()

	_, err := s.Unmarshal([]byte(`{invalid}`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestToJSONSchema tests JSON Schema generation
func TestToJSONSchema(t *testing.T) {
	s, err := NewSchema[SimpleStruct](WithDescription("A simple schema"))
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	jsonSchema, err := s.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema failed: %v", err)
	}

	if jsonSchema["type"] != "object" {
		t.Errorf("expected type 'object', got %v", jsonSchema["type"])
	}

	if jsonSchema["description"] != "A simple schema" {
		t.Errorf("expected description 'A simple schema', got %v", jsonSchema["description"])
	}

	props, ok := jsonSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties to be map[string]any, got %T", jsonSchema["properties"])
	}

	if len(props) != 3 {
		t.Errorf("expected 3 properties, got %d", len(props))
	}

	nameSchema, ok := props["name"].(map[string]any)
	if !ok {
		t.Fatalf("expected name schema to be map[string]any")
	}

	if nameSchema["type"] != string(TypeString) {
		t.Errorf("expected name type 'string', got %v", nameSchema["type"])
	}

	if nameSchema["description"] != "The name" {
		t.Errorf("expected description 'The name', got %v", nameSchema["description"])
	}

	required, ok := jsonSchema["required"].([]string)
	if !ok {
		t.Fatalf("expected required to be []string")
	}

	// name and age are required (no omitempty), email is optional
	if len(required) != 2 {
		t.Errorf("expected 2 required fields, got %d: %v", len(required), required)
	}
}

// TestToJSONSchema_ArrayWithItems tests JSON Schema generation for arrays
func TestToJSONSchema_ArrayWithItems(t *testing.T) {
	s, err := NewSchema[StructWithSlice]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	jsonSchema, err := s.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema failed: %v", err)
	}

	props := jsonSchema["properties"].(map[string]any)
	tagsSchema := props["tags"].(map[string]any)

	if tagsSchema["type"] != string(TypeArray) {
		t.Errorf("expected tags type 'array', got %v", tagsSchema["type"])
	}

	items, ok := tagsSchema["items"].(map[string]any)
	if !ok {
		t.Fatalf("expected items to be map[string]any")
	}

	if items["type"] != string(TypeString) {
		t.Errorf("expected items type 'string', got %v", items["type"])
	}
}

// TestToJSONSchema_NestedObject tests JSON Schema generation for nested objects
func TestToJSONSchema_NestedObject(t *testing.T) {
	s, err := NewSchema[NestedStruct]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	jsonSchema, err := s.ToJSONSchema()
	if err != nil {
		t.Fatalf("ToJSONSchema failed: %v", err)
	}

	props := jsonSchema["properties"].(map[string]any)
	addressSchema := props["address"].(map[string]any)

	if addressSchema["type"] != string(TypeObject) {
		t.Errorf("expected address type 'object', got %v", addressSchema["type"])
	}

	nestedProps, ok := addressSchema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested properties")
	}

	if len(nestedProps) != 2 {
		t.Errorf("expected 2 nested properties, got %d", len(nestedProps))
	}
}

// TestToPromptDescription tests prompt description generation
func TestToPromptDescription(t *testing.T) {
	s, err := NewSchema[SimpleStruct](WithDescription("Extract person information"))
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	desc := s.ToPromptDescription()

	if !strings.Contains(desc, "## Content Type") {
		t.Error("expected '## Content Type' header")
	}

	if !strings.Contains(desc, "Extract person information") {
		t.Error("expected schema description in output")
	}

	if !strings.Contains(desc, "## Fields to Extract") {
		t.Error("expected '## Fields to Extract' header")
	}

	if !strings.Contains(desc, "name (string, required)") {
		t.Error("expected name field with type and required annotation")
	}

	if !strings.Contains(desc, "The name") {
		t.Error("expected field description")
	}
}

// TestToPromptDescription_NoDescription tests default description
func TestToPromptDescription_NoDescription(t *testing.T) {
	s, err := NewSchema[SimpleStruct]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	desc := s.ToPromptDescription()

	if !strings.Contains(desc, "Extract the following structured data") {
		t.Error("expected default description when none provided")
	}
}

// TestToPromptDescription_ArrayWithObjectItems tests prompt for array of objects
func TestToPromptDescription_ArrayWithObjectItems(t *testing.T) {
	s, err := NewSchema[StructWithSlice]()
	if err != nil {
		t.Fatalf("NewSchema failed: %v", err)
	}

	desc := s.ToPromptDescription()

	if !strings.Contains(desc, "ingredients (array") {
		t.Error("expected ingredients field")
	}

	if !strings.Contains(desc, "Each item:") {
		t.Error("expected 'Each item:' for array of objects")
	}
}

// TestGetJSONName tests JSON name extraction from struct tags
func TestGetJSONName(t *testing.T) {
	type TestStruct struct {
		WithTag    string `json:"custom_name"`
		WithOmit   string `json:"with_omit,omitempty"`
		NoTag      string
		DashTag    string `json:"-"`
		EmptyFirst string `json:",omitempty"`
	}

	rt := reflect.TypeOf(TestStruct{})

	tests := []struct {
		fieldName string
		expected  string
	}{
		{"WithTag", "custom_name"},
		{"WithOmit", "with_omit"},
		{"NoTag", "NoTag"},
		{"DashTag", "DashTag"}, // json:"-" returns field name
		{"EmptyFirst", "EmptyFirst"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			sf, _ := rt.FieldByName(tt.fieldName)
			got := getJSONName(sf)
			if got != tt.expected {
				t.Errorf("getJSONName() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestHasOmitempty tests omitempty detection
func TestHasOmitempty(t *testing.T) {
	type TestStruct struct {
		WithOmit    string `json:"name,omitempty"`
		WithoutOmit string `json:"other_name"`
		NoTag       string
	}

	rt := reflect.TypeOf(TestStruct{})

	tests := []struct {
		fieldName string
		expected  bool
	}{
		{"WithOmit", true},
		{"WithoutOmit", false},
		{"NoTag", false},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			sf, _ := rt.FieldByName(tt.fieldName)
			got := hasOmitempty(sf)
			if got != tt.expected {
				t.Errorf("hasOmitempty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestParseValidators tests validator tag parsing
func TestParseValidators(t *testing.T) {
	tests := []struct {
		tag      string
		expected []string
	}{
		{"", nil},
		{"required", []string{"required"}},
		{"required,email", []string{"required", "email"}},
		{"min=2,max=100", []string{"min=2", "max=100"}},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := parseValidators(tt.tag)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseValidators(%q) = %v, want %v", tt.tag, got, tt.expected)
			}
		})
	}
}

// TestValidationError_Error tests ValidationError string formatting
func TestValidationError_Error(t *testing.T) {
	err := ValidationError{
		Field:   "email",
		Message: "must be a valid email address",
		Value:   "invalid",
	}

	expected := "email: must be a valid email address"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

// TestField_MarshalJSON tests Field JSON marshaling
func TestField_MarshalJSON(t *testing.T) {
	field := Field{
		Name:        "address",
		Type:        TypeObject,
		Description: "Address info",
		Properties: []Field{
			{Name: "street", Type: TypeString},
			{Name: "city", Type: TypeString},
		},
	}

	data, err := json.Marshal(field)
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	props, ok := result["properties"].([]any)
	if !ok {
		t.Fatalf("expected properties to be array, got %T", result["properties"])
	}

	if len(props) != 2 {
		t.Errorf("expected 2 properties, got %d", len(props))
	}
}
