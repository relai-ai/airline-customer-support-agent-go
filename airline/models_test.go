package airline

import "testing"

func TestSelectModelPrefersOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "openai-key")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "")

	selection, err := SelectModel()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selection.Provider != "openai" {
		t.Errorf("provider = %q, want openai", selection.Provider)
	}
	if selection.Model != OpenAIDefaultModel {
		t.Errorf("model = %q, want %q", selection.Model, OpenAIDefaultModel)
	}
}

func TestSelectModelUsesAnthropicWhenOpenAIAbsent(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-key")
	t.Setenv("ANTHROPIC_MODEL", "")

	selection, err := SelectModel()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selection.Provider != "anthropic" {
		t.Errorf("provider = %q, want anthropic", selection.Provider)
	}
	if selection.Model != AnthropicDefaultModel {
		t.Errorf("model = %q, want %q", selection.Model, AnthropicDefaultModel)
	}
}

func TestSelectModelPrefersOpenAIWhenBothKeysSet(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "openai-key")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-key")

	selection, err := SelectModel()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selection.Provider != "openai" {
		t.Errorf("provider = %q, want openai", selection.Provider)
	}
}

func TestSelectModelHonorsModelOverride(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "openai-key")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_MODEL", "gpt-4.1-mini")

	selection, err := SelectModel()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selection.Model != "gpt-4.1-mini" {
		t.Errorf("model = %q, want gpt-4.1-mini", selection.Model)
	}
}

func TestSelectModelRequiresAKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")

	if _, err := SelectModel(); err == nil {
		t.Fatal("expected an error when no provider key is set")
	}
}
