package relai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"strings"

	"google.golang.org/genai"

	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
)

const (
	judgeAppName   = "relai-judge"
	judgeUserID    = "relai"
	judgeSessionID = "judge"
)

type judgeModelFactory func(context.Context, ModelSpec) (adkmodel.LLM, error)

func runLLMJudge(ctx context.Context, evaluator LLMJudgeEvaluator, result SimulationResult, factory judgeModelFactory) (EvaluationResult, error) {
	minScore, maxScore, err := resolveScoreBounds(evaluator.MinScore, evaluator.MaxScore)
	if err != nil {
		return EvaluationResult{}, fmt.Errorf("LLM judge %q has invalid score bounds: %w", evaluator.ID(), err)
	}
	output, err := runStructuredModel(ctx, evaluator.Model, "relai_judge", "Scores a RELAI simulation result.", evaluator.Instructions, judgePrompt(evaluator, result), factory)
	if err != nil {
		return EvaluationResult{}, fmt.Errorf("run LLM judge %q: %w", evaluator.ID(), err)
	}

	var evaluation EvaluationResult
	if err := json.Unmarshal([]byte(output), &evaluation); err != nil {
		return EvaluationResult{}, fmt.Errorf("LLM judge %q returned invalid evaluation JSON: %w", evaluator.ID(), err)
	}
	if err := validateEvaluationResult(evaluator.ID(), evaluation, minScore, maxScore); err != nil {
		return EvaluationResult{}, err
	}
	return evaluation, nil
}

func runStructuredModel(ctx context.Context, spec ModelSpec, name, description, instruction, prompt string, factory judgeModelFactory) (string, error) {
	model, err := factory(ctx, spec)
	if err != nil {
		return "", err
	}
	agent, err := llmagent.New(llmagent.Config{
		Name:                  name,
		Description:           description,
		Instruction:           instruction,
		Model:                 model,
		GenerateContentConfig: &genai.GenerateContentConfig{ResponseMIMEType: "application/json"},
	})
	if err != nil {
		return "", fmt.Errorf("create model agent: %w", err)
	}

	sessions := session.InMemoryService()
	if _, err := sessions.Create(ctx, &session.CreateRequest{
		AppName: judgeAppName, UserID: judgeUserID, SessionID: judgeSessionID,
	}); err != nil {
		return "", fmt.Errorf("create model session: %w", err)
	}
	r, err := runner.New(runner.Config{AppName: judgeAppName, Agent: agent, SessionService: sessions})
	if err != nil {
		return "", fmt.Errorf("create model runner: %w", err)
	}

	var output string
	for event, err := range r.Run(ctx, judgeUserID, judgeSessionID, genai.NewContentFromText(prompt, genai.RoleUser), adkagent.RunConfig{}) {
		if err != nil {
			return "", err
		}
		if event != nil && event.IsFinalResponse() {
			output = contentText(event.Content)
		}
	}
	if strings.TrimSpace(output) == "" {
		return "", errors.New("model returned no output")
	}
	return output, nil
}

func judgePrompt(evaluator LLMJudgeEvaluator, result SimulationResult) string {
	minScore, maxScore := defaultScoreBounds(evaluator.MinScore, evaluator.MaxScore)
	events, _ := json.Marshal(result.Events)
	metadata, _ := json.Marshal(result.Metadata)
	finalOutput, _ := json.Marshal(result.FinalOutput)
	return fmt.Sprintf("Evaluate this RELAI simulation result.\n\nEvaluator instructions:\n%s\n\nScore bounds: %v to %v\n\nFinal output:\n%s\n\nStop reason:\n%s\n\nSimulation metadata:\n%s\n\nTranscript events:\n%s\n\nReturn JSON only with numeric score and non-empty feedback.", evaluator.Instructions, minScore, maxScore, finalOutput, result.StopReason, metadata, events)
}

func defaultJudgeModel(ctx context.Context, spec ModelSpec) (adkmodel.LLM, error) {
	provider := strings.ToLower(strings.TrimSpace(spec.Provider))
	if provider == "google" || provider == "gemini" {
		apiKey := providerAPIKey("google", "GOOGLE_API_KEY", "GEMINI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("LLM judge model %q (provider=%q) is missing provider configuration: set RELAI_MODEL_PROVIDER_GOOGLE_API_KEY, GOOGLE_API_KEY, or GEMINI_API_KEY", spec.Name, spec.Provider)
		}
		model, err := gemini.NewModel(ctx, spec.Name, &genai.ClientConfig{APIKey: apiKey})
		if err != nil {
			return nil, fmt.Errorf("configure Gemini judge model %q: %w", spec.Name, err)
		}
		return model, nil
	}
	return newOpenAICompatibleModel(spec)
}

func providerAPIKey(provider string, fallback ...string) string {
	prefix := strings.ToUpper(strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(provider))
	for _, key := range append([]string{"RELAI_MODEL_PROVIDER_" + prefix + "_API_KEY", prefix + "_API_KEY"}, fallback...) {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

type openAICompatibleModel struct {
	name    string
	baseURL string
	apiKey  string
	client  *http.Client
}

func newOpenAICompatibleModel(spec ModelSpec) (adkmodel.LLM, error) {
	provider := strings.ToLower(strings.TrimSpace(spec.Provider))
	if provider == "" {
		provider = "openai"
	}
	apiKey := providerAPIKey(provider)
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("RELAI_MODEL_PROVIDER_"+strings.ToUpper(strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(provider))+"_BASE_URL")), "/")
	if provider == "openai" && baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	missing := []string{}
	if apiKey == "" {
		missing = append(missing, "RELAI_MODEL_PROVIDER_"+strings.ToUpper(strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(provider))+"_API_KEY or "+strings.ToUpper(strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(provider))+"_API_KEY")
	}
	if baseURL == "" {
		missing = append(missing, "RELAI_MODEL_PROVIDER_"+strings.ToUpper(strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(provider))+"_BASE_URL")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("LLM judge model %q (provider=%q) is missing provider configuration: set %s", spec.Name, spec.Provider, strings.Join(missing, "; set "))
	}
	return &openAICompatibleModel{name: spec.Name, baseURL: baseURL, apiKey: apiKey, client: http.DefaultClient}, nil
}

func (m *openAICompatibleModel) Name() string { return m.name }

func (m *openAICompatibleModel) GenerateContent(ctx context.Context, req *adkmodel.LLMRequest, _ bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		response, err := m.generate(ctx, req)
		yield(response, err)
	}
}

func (m *openAICompatibleModel) generate(ctx context.Context, req *adkmodel.LLMRequest) (*adkmodel.LLMResponse, error) {
	payload := openAIChatRequest{Model: m.name, ResponseFormat: &openAIResponseFormat{Type: "json_object"}, Messages: openAIMessages(req)}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Authorization", "Bearer "+m.apiKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := m.client.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("OpenAI-compatible provider returned %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}
	var decoded openAIChatResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, err
	}
	if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return nil, fmt.Errorf("OpenAI-compatible provider returned no message content")
	}
	return &adkmodel.LLMResponse{Content: genai.NewContentFromText(decoded.Choices[0].Message.Content, genai.RoleModel)}, nil
}

type openAIChatRequest struct {
	Model          string                `json:"model"`
	Messages       []openAIChatMessage   `json:"messages"`
	ResponseFormat *openAIResponseFormat `json:"response_format,omitempty"`
}

type openAIResponseFormat struct {
	Type string `json:"type"`
}
type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
}

func openAIMessages(req *adkmodel.LLMRequest) []openAIChatMessage {
	messages := []openAIChatMessage{}
	if req.Config != nil && req.Config.SystemInstruction != nil {
		if text := contentText(req.Config.SystemInstruction); text != "" {
			messages = append(messages, openAIChatMessage{Role: "system", Content: text})
		}
	}
	for _, content := range req.Contents {
		text := contentText(content)
		if text == "" {
			continue
		}
		role := "user"
		if content.Role == genai.RoleModel {
			role = "assistant"
		}
		messages = append(messages, openAIChatMessage{Role: role, Content: text})
	}
	return messages
}

func contentText(content *genai.Content) string {
	if content == nil {
		return ""
	}
	parts := make([]string, 0, len(content.Parts))
	for _, part := range content.Parts {
		if part != nil && part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	return strings.Join(parts, "")
}
