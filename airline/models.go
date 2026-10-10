package airline

import (
	"fmt"
	"os"
)

// Default models used when the matching *_MODEL override is not set.
const (
	OpenAIDefaultModel    = "gpt-4.1"
	AnthropicDefaultModel = "claude-sonnet-5"
)

// ModelSelection describes the runtime provider and model to use.
type ModelSelection struct {
	Provider string
	Model    string
}

// SelectModel selects the runtime model provider from environment keys.
//
// OpenAI is preferred when both provider keys are present, mirroring the
// Python and TypeScript versions of this agent. The concrete model can be
// overridden with OPENAI_MODEL / ANTHROPIC_MODEL.
func SelectModel() (ModelSelection, error) {
	if os.Getenv("AIRLINE_MODEL_PROVIDER") == "relai" {
		return ModelSelection{Provider: "relai", Model: "relai-session-profile"}, nil
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		return ModelSelection{
			Provider: "openai",
			Model:    envOr("OPENAI_MODEL", OpenAIDefaultModel),
		}, nil
	}

	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return ModelSelection{
			Provider: "anthropic",
			Model:    envOr("ANTHROPIC_MODEL", AnthropicDefaultModel),
		}, nil
	}

	return ModelSelection{}, fmt.Errorf(
		"set OPENAI_API_KEY or ANTHROPIC_API_KEY before starting the agent",
	)
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
