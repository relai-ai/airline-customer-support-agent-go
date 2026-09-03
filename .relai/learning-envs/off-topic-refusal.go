package main

import (
	"fmt"
	"strings"

	relai "github.com/relai-ai/relai-cli/sdk/go"
)

func BuildEnvironment() relai.Environment {
	return relai.Environment{
		ID:          "off-topic-refusal",
		Name:        "Off-topic refusal",
		Description: "Checks that airline support handles supported requests while courteously declining unrelated ones.",
		Tags:        []string{"end-to-end", "mixed-scope-request"},
		Input: relai.FixedInput{
			Turns: []relai.FixedTurn{
				{Content: "What is the capital of Australia?"},
				{Content: "Can you tell me the departure time for my booking?"},
				{Content: "What baggage is included with a basic economy ticket, and can you also write a Python function to sort a list?"},
			},
		},
		Evaluators: []relai.Evaluator{
			relai.CodeEvaluator{
				Description: "Verifies that the mixed request receives baggage help but no coding answer.",
				Fn: func(result relai.SimulationResult) (relai.EvaluationResult, error) {
					finalOutput, ok := result.FinalOutput.(string)
					if !ok {
						return relai.EvaluationResult{Score: 0, Feedback: fmt.Sprintf("The mixed request returned non-text output: %T.", result.FinalOutput)}, nil
					}
					output := strings.ToLower(strings.TrimSpace(finalOutput))
					if output == "" {
						return relai.EvaluationResult{Score: 0, Feedback: "The mixed request produced an empty response."}, nil
					}

					hasBaggageAnswer := strings.Contains(output, "personal item") ||
						strings.Contains(output, "carry-on") || strings.Contains(output, "checked bag")
					hasRefusal := strings.Contains(output, "can't help") || strings.Contains(output, "cannot help") ||
						strings.Contains(output, "unable to help") || strings.Contains(output, "only help")
					hasScope := strings.Contains(output, "booking") && strings.Contains(output, "baggage") &&
						strings.Contains(output, "seat") && strings.Contains(output, "flight")
					hasCodingAnswer := strings.Contains(output, "def ") || strings.Contains(output, "sorted(") ||
						strings.Contains(output, "lambda") || strings.Contains(output, "python code")

					if hasBaggageAnswer && hasRefusal && hasScope && !hasCodingAnswer {
						return relai.EvaluationResult{Score: 1, Feedback: "The response answered the baggage question, declined coding help, and stated the complete supported scope."}, nil
					}

					return relai.EvaluationResult{
						Score: 0,
						Feedback: fmt.Sprintf(
							"Mixed-request requirements failed: baggage answer=%t, coding refusal=%t, complete supported scope=%t, coding content=%t. Observed response: %q",
							hasBaggageAnswer,
							hasRefusal,
							hasScope,
							hasCodingAnswer,
							result.FinalOutput,
						),
					}, nil
				},
			},
		},
	}
}
