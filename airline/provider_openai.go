package airline

import (
	"context"
	"encoding/json"
	"io"

	"github.com/openai/openai-go/v3"
)

type openaiRunner struct {
	client openai.Client
	model  string
}

func newOpenAIRunner(model string) *openaiRunner {
	return &openaiRunner{client: openai.NewClient(), model: model}
}

func (r *openaiRunner) run(
	ctx context.Context,
	instructions string,
	tools []Tool,
	history []Message,
	out io.Writer,
) (string, error) {
	toolIndex := make(map[string]Tool, len(tools))
	toolParams := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		toolIndex[t.Name] = t
		toolParams = append(toolParams, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  openai.FunctionParameters(t.Parameters),
		}))
	}

	messages := []openai.ChatCompletionMessageParamUnion{openai.SystemMessage(instructions)}
	for _, m := range history {
		switch m.Role {
		case "user":
			messages = append(messages, openai.UserMessage(m.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(m.Content))
		}
	}

	for {
		stream := r.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
			Model:    openai.ChatModel(r.model),
			Messages: messages,
			Tools:    toolParams,
		})

		acc := openai.ChatCompletionAccumulator{}
		for stream.Next() {
			chunk := stream.Current()
			acc.AddChunk(chunk)
			if len(chunk.Choices) > 0 {
				if delta := chunk.Choices[0].Delta.Content; delta != "" {
					if _, err := io.WriteString(out, delta); err != nil {
						return "", err
					}
				}
			}
		}
		if err := stream.Err(); err != nil {
			return "", err
		}
		if len(acc.Choices) == 0 {
			return "", nil
		}

		message := acc.Choices[0].Message
		if len(message.ToolCalls) == 0 {
			return message.Content, nil
		}

		messages = append(messages, message.ToParam())
		for _, call := range message.ToolCalls {
			var args map[string]any
			if call.Function.Arguments != "" {
				_ = json.Unmarshal([]byte(call.Function.Arguments), &args)
			}
			result := "Unknown tool."
			if tool, ok := toolIndex[call.Function.Name]; ok {
				result = tool.Execute(args)
			}
			messages = append(messages, openai.ToolMessage(result, call.ID))
		}
	}
}
