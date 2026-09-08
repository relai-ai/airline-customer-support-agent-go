package main

import (
	"fmt"
	"strings"
	"unicode"

	relai "github.com/relai-ai/relai-cli/sdk/go"
)

func BuildEnvironment() relai.Environment {
	const requiredSignoff = "Please let me know if you have any questions."
	const normalizedSignoff = "please let me know if you have any questions"

	return relai.Environment{
		ID:          "response-signoff",
		Name:        "response-signoff",
		Description: "Checks that the agent ends its reply with the required signoff.",
		Tags:        []string{"end-to-end", "signoff"},
		Target:      relai.AgentTarget{},
		Input: relai.FixedInput{
			Turns: []relai.FixedTurn{{
				Content: "What's the baggage policy for a standard ticket?",
			}},
		},
		Evaluators: []relai.Evaluator{
			relai.CodeEvaluator{
				EvaluatorID: "required-signoff",
				Description: "Verifies that the final response ends with the required signoff, allowing punctuation and casing variation.",
				Fn: func(result relai.SimulationResult) (relai.EvaluationResult, error) {
					finalText, ok := result.FinalOutput.(string)
					if !ok {
						return relai.EvaluationResult{
							Score:    0,
							Feedback: fmt.Sprintf("Expected final output to be a string ending with %q, but observed %T.", requiredSignoff, result.FinalOutput),
						}, nil
					}

					normalizedOutput := normalizeEnding(finalText)
					if strings.HasSuffix(normalizedOutput, normalizedSignoff) {
						return relai.EvaluationResult{
							Score:    1,
							Feedback: "The final response ends with the required signoff.",
						}, nil
					}

					observedEnding := finalText
					if len(observedEnding) > 120 {
						observedEnding = observedEnding[len(observedEnding)-120:]
					}

					return relai.EvaluationResult{
						Score: 0,
						Feedback: fmt.Sprintf(
							"The final response does not end with the required signoff. Expected an ending equivalent to %q, but the observed ending was %q.",
							requiredSignoff,
							observedEnding,
						),
					}, nil
				},
			},
		},
	}
}

func normalizeEnding(text string) string {
	trimmed := strings.TrimSpace(text)
	trimmed = strings.TrimRightFunc(trimmed, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	return strings.ToLower(trimmed)
}
