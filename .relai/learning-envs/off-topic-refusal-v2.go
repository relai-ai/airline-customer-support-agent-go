package main

import (
	relai "github.com/relai-ai/relai-cli/sdk/go"
)

const judgeModel = "gpt-4.1"

func BuildEnvironment() relai.Environment {
	return relai.Environment{
		ID:          "off-topic-refusal-v2",
		Name:        "Off-topic refusal v2",
		Description: "Checks that SkyServe handles airline support requests while courteously declining unrelated requests.",
		Tags:        []string{"end-to-end", "mixed-scope-request"},
		Input: relai.FixedInput{
			Turns: []relai.FixedTurn{
				{Content: "Can you book me a hotel in Seattle for tomorrow night?"},
				{Content: "What time does my flight depart?"},
				{Content: "My confirmation code is SKY123."},
				{Content: "What baggage is included with a basic economy ticket, and what is the capital of Australia?"},
			},
		},
		Evaluators: []relai.Evaluator{
			relai.LLMJudgeEvaluator{
				EvaluatorID:  "standalone-off-topic-refusal",
				Description:  "Checks the standalone off-topic refusal.",
				Model:        relai.ModelSpec{Provider: "openai", Name: judgeModel},
				Instructions: `Evaluate only transcript turn 0, where the user asks SkyServe to book a hotel. Score 1 only if the assistant courteously and briefly declines the hotel request, does not provide hotel booking help, recommendations, prices, availability, or any partial answer, and clearly states that it can only help with all four supported topics: booking lookups, baggage policy, seat changes, and flight changes. It may invite an on-topic question. Score 0 if it answers or speculates about hotels, omits any supported topic from its scope statement, gives an unclear/non-refusal response, or is rude. In feedback, state the decisive pass/fail reason and quote the relevant observed behavior.`,
			},
			relai.LLMJudgeEvaluator{
				EvaluatorID:  "booking-confirmation-code-handling",
				Description:  "Checks helpful booking handling and confirmation-code verification.",
				Model:        relai.ModelSpec{Provider: "openai", Name: judgeModel},
				Instructions: `Evaluate only transcript turns 1 and 2 as one booking-lookup interaction. On turn 1, the user asks for a specific flight's departure time without a code; the assistant must ask for a confirmation code rather than refuse a legitimate airline request or invent a departure time. On turn 2, the user provides SKY123; the assistant must use or otherwise verify that code and helpfully provide the booking departure information. Score 1 only if both conditions hold. Score 0 if it over-refuses either on-topic request, fails to request a code before discussing the specific booking, invents an unverified booking fact, or fails to provide helpful departure information after the code. In feedback, identify the decisive pass/fail behavior for both turns.`,
			},
			relai.LLMJudgeEvaluator{
				EvaluatorID:  "mixed-scope-request-handling",
				Description:  "Checks that a mixed airline and off-topic request is split correctly.",
				Model:        relai.ModelSpec{Provider: "openai", Name: judgeModel},
				Instructions: `Evaluate only transcript turn 3. The user asks both about basic-economy baggage and the capital of Australia. Score 1 only if the assistant answers the baggage-policy portion helpfully without inventing fees or exceptions, while courteously declining the Australia/trivia portion without giving, hinting at, or speculating about the capital. The refusal must state that it can only help with all four supported topics: booking lookups, baggage policy, seat changes, and flight changes. Score 0 if it answers any trivia, omits the baggage answer, over-refuses the valid baggage question, omits any supported topic from its scope statement, or invents baggage details. In feedback, state the decisive pass/fail reason and distinguish airline handling from off-topic handling.`,
			},
		},
	}
}
