package inference

import (
	"testing"
	"time"
)

// --- Role constants ---

func TestRoleConstants(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		expected string
	}{
		{name: "RoleSystem", role: RoleSystem, expected: "system"},
		{name: "RoleUser", role: RoleUser, expected: "user"},
		{name: "RoleAssistant", role: RoleAssistant, expected: "assistant"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.role) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.role))
			}
		})
	}
}

func TestRoleConstants_AreDistinct(t *testing.T) {
	roles := []Role{RoleSystem, RoleUser, RoleAssistant}
	seen := make(map[Role]bool)
	for _, r := range roles {
		if seen[r] {
			t.Errorf("duplicate role constant: %q", r)
		}
		seen[r] = true
	}
}

// --- Message struct ---

func TestMessage_Construction(t *testing.T) {
	msg := Message{
		Role:         RoleUser,
		Content:      "Hello, world!",
		CacheControl: "ephemeral",
	}

	if msg.Role != RoleUser {
		t.Errorf("Role: expected %q, got %q", RoleUser, msg.Role)
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("Content: expected %q, got %q", "Hello, world!", msg.Content)
	}
	if msg.CacheControl != "ephemeral" {
		t.Errorf("CacheControl: expected %q, got %q", "ephemeral", msg.CacheControl)
	}
}

func TestMessage_ZeroValue(t *testing.T) {
	var msg Message
	if msg.Role != "" {
		t.Errorf("zero Role: expected empty, got %q", msg.Role)
	}
	if msg.Content != "" {
		t.Errorf("zero Content: expected empty, got %q", msg.Content)
	}
	if msg.CacheControl != "" {
		t.Errorf("zero CacheControl: expected empty, got %q", msg.CacheControl)
	}
}

// --- Request struct ---

func TestRequest_Construction(t *testing.T) {
	schema := map[string]any{"type": "object"}
	req := Request{
		Messages: []Message{
			{Role: RoleSystem, Content: "You are helpful."},
			{Role: RoleUser, Content: "Extract data."},
		},
		MaxTokens:   1024,
		Temperature: 0.7,
		Grammar:     "root ::= text",
		JSONSchema:  schema,
		StrictMode:  true,
	}

	if len(req.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(req.Messages))
	}
	if req.Messages[0].Role != RoleSystem {
		t.Errorf("first message role: expected %q, got %q", RoleSystem, req.Messages[0].Role)
	}
	if req.Messages[1].Content != "Extract data." {
		t.Errorf("second message content: expected %q, got %q", "Extract data.", req.Messages[1].Content)
	}
	if req.MaxTokens != 1024 {
		t.Errorf("MaxTokens: expected 1024, got %d", req.MaxTokens)
	}
	if req.Temperature != 0.7 {
		t.Errorf("Temperature: expected 0.7, got %f", req.Temperature)
	}
	if req.Grammar != "root ::= text" {
		t.Errorf("Grammar: expected %q, got %q", "root ::= text", req.Grammar)
	}
	if req.JSONSchema["type"] != "object" {
		t.Errorf("JSONSchema[\"type\"]: expected %q, got %v", "object", req.JSONSchema["type"])
	}
	if !req.StrictMode {
		t.Error("StrictMode: expected true, got false")
	}
}

func TestRequest_ZeroValue(t *testing.T) {
	var req Request
	if req.Messages != nil {
		t.Error("zero Messages: expected nil")
	}
	if req.MaxTokens != 0 {
		t.Errorf("zero MaxTokens: expected 0, got %d", req.MaxTokens)
	}
	if req.Temperature != 0 {
		t.Errorf("zero Temperature: expected 0, got %f", req.Temperature)
	}
	if req.Grammar != "" {
		t.Errorf("zero Grammar: expected empty, got %q", req.Grammar)
	}
	if req.JSONSchema != nil {
		t.Error("zero JSONSchema: expected nil")
	}
	if req.StrictMode {
		t.Error("zero StrictMode: expected false")
	}
}

// --- Usage struct ---

func TestUsage_Construction(t *testing.T) {
	usage := Usage{
		InputTokens:  100,
		OutputTokens: 50,
	}

	if usage.InputTokens != 100 {
		t.Errorf("InputTokens: expected 100, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 50 {
		t.Errorf("OutputTokens: expected 50, got %d", usage.OutputTokens)
	}
}

func TestUsage_ZeroValue(t *testing.T) {
	var usage Usage
	if usage.InputTokens != 0 {
		t.Errorf("zero InputTokens: expected 0, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 0 {
		t.Errorf("zero OutputTokens: expected 0, got %d", usage.OutputTokens)
	}
}

// --- Response struct ---

func TestResponse_Construction(t *testing.T) {
	resp := Response{
		Content:      "extracted data",
		Usage:        Usage{InputTokens: 200, OutputTokens: 100},
		Model:        "gpt-4o",
		Provider:     "openai",
		Duration:     2 * time.Second,
		FinishReason: "stop",
		GenerationID: "gen-abc123",
		Cost:         0.005,
		CostIncluded: true,
	}

	if resp.Content != "extracted data" {
		t.Errorf("Content: expected %q, got %q", "extracted data", resp.Content)
	}
	if resp.Usage.InputTokens != 200 {
		t.Errorf("Usage.InputTokens: expected 200, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 100 {
		t.Errorf("Usage.OutputTokens: expected 100, got %d", resp.Usage.OutputTokens)
	}
	if resp.Model != "gpt-4o" {
		t.Errorf("Model: expected %q, got %q", "gpt-4o", resp.Model)
	}
	if resp.Provider != "openai" {
		t.Errorf("Provider: expected %q, got %q", "openai", resp.Provider)
	}
	if resp.Duration != 2*time.Second {
		t.Errorf("Duration: expected %v, got %v", 2*time.Second, resp.Duration)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason: expected %q, got %q", "stop", resp.FinishReason)
	}
	if resp.GenerationID != "gen-abc123" {
		t.Errorf("GenerationID: expected %q, got %q", "gen-abc123", resp.GenerationID)
	}
	if resp.Cost != 0.005 {
		t.Errorf("Cost: expected 0.005, got %f", resp.Cost)
	}
	if !resp.CostIncluded {
		t.Error("CostIncluded: expected true, got false")
	}
}

func TestResponse_ZeroValue(t *testing.T) {
	var resp Response
	if resp.Content != "" {
		t.Errorf("zero Content: expected empty, got %q", resp.Content)
	}
	if resp.Model != "" {
		t.Errorf("zero Model: expected empty, got %q", resp.Model)
	}
	if resp.Provider != "" {
		t.Errorf("zero Provider: expected empty, got %q", resp.Provider)
	}
	if resp.Duration != 0 {
		t.Errorf("zero Duration: expected 0, got %v", resp.Duration)
	}
	if resp.FinishReason != "" {
		t.Errorf("zero FinishReason: expected empty, got %q", resp.FinishReason)
	}
	if resp.GenerationID != "" {
		t.Errorf("zero GenerationID: expected empty, got %q", resp.GenerationID)
	}
	if resp.Cost != 0 {
		t.Errorf("zero Cost: expected 0, got %f", resp.Cost)
	}
	if resp.CostIncluded {
		t.Error("zero CostIncluded: expected false")
	}
}

func TestResponse_FinishReason_Values(t *testing.T) {
	tests := []struct {
		name   string
		reason string
	}{
		{name: "stop", reason: "stop"},
		{name: "length", reason: "length"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := Response{FinishReason: tt.reason}
			if resp.FinishReason != tt.reason {
				t.Errorf("expected %q, got %q", tt.reason, resp.FinishReason)
			}
		})
	}
}
