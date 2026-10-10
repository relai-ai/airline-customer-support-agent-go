package airline

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

const anthropicMaxTokens = 1024

type anthropicRunner struct {
	client anthropic.Client
	model  string
}

func newAnthropicRunner(model string) *anthropicRunner {
	return &anthropicRunner{client: anthropic.NewClient(), model: model}
}

func (r *anthropicRunner) run(
	ctx context.Context,
	instructions string,
	tools []Tool,
	history []Message,
	out io.Writer,
) (string, error) {
	toolIndex := make(map[string]Tool, len(tools))
	toolParams := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		toolIndex[t.Name] = t
		properties, _ := t.Parameters["properties"].(map[string]any)
		toolParams = append(toolParams, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: anthropic.ToolInputSchemaParam{Properties: properties},
			},
		})
	}

	var messages []anthropic.MessageParam
	for _, m := range history {
		switch m.Role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		}
	}

	system := []anthropic.TextBlockParam{{Text: instructions}}

	for {
		stream := r.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
			Model:     anthropic.Model(r.model),
			MaxTokens: anthropicMaxTokens,
			System:    system,
			Messages:  messages,
			Tools:     toolParams,
		})

		message := anthropic.Message{}
		for stream.Next() {
			event := stream.Current()
			if err := message.Accumulate(event); err != nil {
				return "", err
			}
			if delta, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
				if text, ok := delta.Delta.AsAny().(anthropic.TextDelta); ok {
					if _, err := io.WriteString(out, text.Text); err != nil {
						return "", err
					}
				}
			}
		}
		if err := stream.Err(); err != nil {
			return "", err
		}

		trace := recorder(ctx)
		step := -1
		if trace != nil {
			var calls []map[string]any
			for _, block := range message.Content {
				if use, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
					calls = append(calls, map[string]any{"tool_call_id": use.ID, "function_name": use.Name, "arguments": observedArguments(string(use.Input))})
				}
			}
			step = trace.model(anthropicText(message), r.model, calls)
		}
		messages = append(messages, message.ToParam())

		var toolResults []anthropic.ContentBlockParamUnion
		for _, block := range message.Content {
			if use, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				var args map[string]any
				if len(use.Input) > 0 {
					_ = json.Unmarshal(use.Input, &args)
				}
				result := "Unknown tool."
				if tool, ok := toolIndex[use.Name]; ok {
					result = tool.Execute(args)
				}
				trace.result(step, use.ID, result)
				toolResults = append(toolResults, anthropic.NewToolResultBlock(use.ID, result, false))
			}
		}

		if len(toolResults) == 0 {
			return anthropicText(message), nil
		}
		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}
}

func anthropicText(message anthropic.Message) string {
	var builder strings.Builder
	for _, block := range message.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			builder.WriteString(text.Text)
		}
	}
	return builder.String()
}
