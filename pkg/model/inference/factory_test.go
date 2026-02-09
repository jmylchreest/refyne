package inference

import (
	"strings"
	"testing"
)

// --- AvailableProviders ---

func TestAvailableProviders_ContainsLocal(t *testing.T) {
	providers := AvailableProviders()

	found := false
	for _, p := range providers {
		if p == "local" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("AvailableProviders() should contain \"local\", got %v", providers)
	}
}

func TestAvailableProviders_ReturnsNonEmpty(t *testing.T) {
	providers := AvailableProviders()
	if len(providers) == 0 {
		t.Error("AvailableProviders() should return at least one provider")
	}
}

func TestAvailableProviders_ContainsRemoteProviders(t *testing.T) {
	providers := AvailableProviders()
	providerSet := make(map[string]bool, len(providers))
	for _, p := range providers {
		providerSet[p] = true
	}

	// The llm package registers these providers; they should appear alongside "local".
	expectedRemote := []string{"anthropic", "openai", "openrouter", "ollama", "helicone"}
	for _, name := range expectedRemote {
		if !providerSet[name] {
			t.Errorf("AvailableProviders() should contain %q, got %v", name, providers)
		}
	}
}

// --- HasAPIKey ---

func TestHasAPIKey_ReturnsFalseWithoutEnvVars(t *testing.T) {
	// These providers require env vars to be set. In a test environment
	// without those vars, HasAPIKey should return false.
	providers := []string{"openrouter", "anthropic", "openai", "cerebras", "helicone"}
	for _, p := range providers {
		t.Run(p, func(t *testing.T) {
			// We clear the env var to be safe, then restore.
			envKeys := map[string]string{
				"openrouter": "OPENROUTER_API_KEY",
				"anthropic":  "ANTHROPIC_API_KEY",
				"openai":     "OPENAI_API_KEY",
				"cerebras":   "CEREBRAS_API_KEY",
				"helicone":   "HELICONE_API_KEY",
			}
			envKey := envKeys[p]
			t.Setenv(envKey, "")

			if HasAPIKey(p) {
				t.Errorf("HasAPIKey(%q) should return false when %s is empty", p, envKey)
			}
		})
	}
}

func TestHasAPIKey_ReturnsTrueWithEnvVar(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "sk-test-key-12345")

	if !HasAPIKey("openrouter") {
		t.Error("HasAPIKey(\"openrouter\") should return true when OPENROUTER_API_KEY is set")
	}
}

func TestHasAPIKey_ReturnsFalseForUnknownProvider(t *testing.T) {
	if HasAPIKey("unknown-provider") {
		t.Error("HasAPIKey(\"unknown-provider\") should return false")
	}
}

func TestHasAPIKey_ReturnsFalseForLocal(t *testing.T) {
	// "local" is not in the envKeys map, so HasAPIKey should return false.
	if HasAPIKey("local") {
		t.Error("HasAPIKey(\"local\") should return false")
	}
}

// --- New ---

func TestNew_WithLocal_ReturnsError(t *testing.T) {
	_, err := New("local")
	if err == nil {
		t.Fatal("New(\"local\") should return an error")
	}
	if !strings.Contains(err.Error(), "NewLocal") {
		t.Errorf("New(\"local\") error should suggest NewLocal(), got: %v", err)
	}
}

func TestNew_WithLocal_ErrorMessage(t *testing.T) {
	_, err := New("local")
	if err == nil {
		t.Fatal("New(\"local\") should return an error")
	}
	// Verify the error message is helpful.
	errMsg := err.Error()
	if !strings.Contains(errMsg, "local") {
		t.Errorf("error message should mention \"local\", got: %q", errMsg)
	}
}
