// Command airline-support runs the SkyServe airline customer support agent as
// an interactive terminal chat. It is the entrypoint invoked by start.sh:
//
//	go run ./cmd/airline-support [log-name]
//
// With no argument a fresh logs/session-<id>.jsonl file is created. Passing a
// log name (e.g. "off-topic-guardrail") writes to a fixed logs/<name>.jsonl.
// Each turn is streamed to the terminal and appended to the session log in the
// JSONL format shared with the Python and TypeScript versions of this agent.
package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	airline "github.com/relai-ai/airline-customer-support-agent-go/airline"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "airline-support: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	selection, err := airline.SelectModel()
	if err != nil {
		return err
	}

	agent, err := airline.NewAgent()
	if err != nil {
		return err
	}

	session, err := openSession(args)
	if err != nil {
		return err
	}
	logPath, err := airline.SessionPath(session.ID)
	if err != nil {
		return err
	}

	fmt.Printf("%s (%s/%s)\n", airline.AgentName, selection.Provider, selection.Model)
	fmt.Printf("Session log: %s\n", logPath)
	fmt.Println("Type exit, quit, or q to end the chat.")

	ctx := context.Background()
	var history []airline.Message

	input := bufio.NewScanner(os.Stdin)
	input.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		fmt.Print("\nYou: ")
		if !input.Scan() {
			fmt.Println()
			break // EOF (Ctrl-D)
		}
		line := strings.TrimSpace(input.Text())
		if line == "" {
			continue
		}
		switch strings.ToLower(line) {
		case "exit", "quit", "q":
			goto done
		}

		if _, err := airline.AppendMessage(session.ID, "user", line); err != nil {
			return err
		}
		history = append(history, airline.Message{Role: "user", Content: line})

		fmt.Print("\nAgent: ")
		var streamed bytes.Buffer
		final, err := agent.StreamResponse(ctx, history, io.MultiWriter(os.Stdout, &streamed))
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nerror: %v\n", err)
			// Drop the unanswered user turn so history stays consistent.
			history = history[:len(history)-1]
			continue
		}
		printReplyTail(streamed.String(), final)

		history = append(history, airline.Message{Role: "assistant", Content: final})
		if _, err := airline.AppendMessage(session.ID, "assistant", final); err != nil {
			return err
		}
	}

done:
	if err := input.Err(); err != nil {
		return err
	}
	fmt.Printf("\nSession saved to %s\n", logPath)
	return nil
}

// openSession creates the log file for this run. A log-name argument maps to a
// fixed logs/<name>.jsonl (overwritten on re-run); otherwise a new random
// session id is generated.
func openSession(args []string) (airline.Session, error) {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		id, err := airline.SessionIDFromLogName(args[0])
		if err != nil {
			return airline.Session{}, err
		}
		return airline.CreateSession(id, true)
	}
	return airline.CreateSession("", false)
}

// printReplyTail makes the on-screen reply match the logged final text. The
// runner streams the raw model output; StreamResponse may append the required
// support sign-off that was never streamed, so print only that missing tail.
func printReplyTail(streamed, final string) {
	shown := strings.TrimRight(streamed, " \t\r\n")
	switch {
	case final == shown:
		// Nothing appended.
	case strings.HasPrefix(final, shown):
		fmt.Print(final[len(shown):])
	default:
		fmt.Print("\n" + final)
	}
	fmt.Println()
}
