package main

import (
 "bufio"
 "context"
 "encoding/json"
 "fmt"
 "io"
 "os"
 "os/signal"
 "syscall"

 airline "github.com/relai-ai/airline-customer-support-agent-go/airline"
)

type request struct {
 ScenarioID string `json:"scenario_id"`
 TurnIndex int `json:"turn_index"`
 UserInput string `json:"user_input"`
 Services json.RawMessage `json:"services"`
 History []struct { Role string `json:"role"`; Content string `json:"content"` } `json:"history"`
 Task json.RawMessage `json:"task"`
}

func run() error {
 ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
 defer cancel()
 agent, err := airline.NewAgent()
 if err != nil { return err }
 scanner := bufio.NewScanner(os.Stdin)
 scanner.Buffer(make([]byte, 65536), 16*1024*1024)
 writer := bufio.NewWriter(os.Stdout)
 encoder := json.NewEncoder(writer)
 for scanner.Scan() {
  var req request
  if err := json.Unmarshal(scanner.Bytes(), &req); err != nil { return err }
  history := make([]airline.Message, 0, len(req.History)+1)
  for _, m := range req.History {
   history = append(history, airline.Message{Role:m.Role, Content:m.Content})
  }
  history = append(history, airline.Message{Role:"user", Content:req.UserInput})
  // Original factory and execution boundary. Its recorder writes synchronously
  // under one lock, so even failures/cancellation leave no queued snapshots.
  reply, err := agent.StreamResponse(ctx, history, io.Discard)
  if err != nil { return err }
  if err := encoder.Encode(map[string]string{"assistant_message":reply}); err != nil { return err }
  if err := writer.Flush(); err != nil { return err }
 }
 return scanner.Err()
}
func main() {
 if err := run(); err != nil {
  // Avoid printing provider error bodies which may contain credentials.
  fmt.Fprintln(os.Stderr, "Airline adapter failed; check provider configuration and connectivity.")
  os.Exit(1)
 }
}
