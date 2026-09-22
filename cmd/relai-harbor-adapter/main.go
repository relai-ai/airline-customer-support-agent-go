package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/relai-ai/airline-customer-support-agent-go/airline"
)

type request struct {
	UserInput string            `json:"user_input"`
	History   []airline.Message `json:"history"`
}

type response struct {
	AssistantMessage string `json:"assistant_message"`
}

func main() {
	var input request
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fail(err)
	}
	if input.UserInput == "" {
		fail(fmt.Errorf("user_input must be a non-empty string"))
	}
	history := input.History
	history = append(history, airline.Message{Role: "user", Content: input.UserInput})
	agent, err := airline.NewAgent()
	if err != nil {
		fail(err)
	}
	message, err := agent.StreamResponse(context.Background(), history, io.Discard)
	if err != nil {
		fail(err)
	}
	if err := json.NewEncoder(os.Stdout).Encode(response{AssistantMessage: message}); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
