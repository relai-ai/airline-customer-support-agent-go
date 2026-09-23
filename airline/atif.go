package airline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// atifRecorder is inert unless Harbor asks for a trajectory. It records only
// provider responses and tool calls/results observed at the real dispatch edge.
type atifRecorder struct {
	path    string
	mu      sync.Mutex
	steps   []atifStep
	pending []atifToolCall
	results map[string]string
}

type atifDocument struct {
	SchemaVersion string     `json:"schema_version"`
	Agent         atifAgent  `json:"agent"`
	Steps         []atifStep `json:"steps"`
}

type atifAgent struct { Name string `json:"name"`; Version string `json:"version"` }
type atifStep struct {
	StepID int `json:"step_id"`
	Source string `json:"source"`
	Message string `json:"message"`
	ToolCalls []atifToolCall `json:"tool_calls,omitempty"`
	Observation *atifObservation `json:"observation,omitempty"`
}
type atifToolCall struct { ToolCallID string `json:"tool_call_id"`; FunctionName string `json:"function_name"`; Arguments map[string]any `json:"arguments"` }
type atifObservation struct { Results []atifToolResult `json:"results"` }
type atifToolResult struct { SourceCallID string `json:"source_call_id"`; Content string `json:"content"` }

func newATIFRecorder() *atifRecorder {
	path := os.Getenv("RELAI_ATIF_TRAJECTORY_PATH")
	if path == "" { return nil }
	return &atifRecorder{path: path, results: make(map[string]string)}
}

// toolCall writes a valid cumulative snapshot before dispatch. Calls remain
// pending until a real, nonempty assistant message permits an ATIF agent step.
func (r *atifRecorder) toolCall(id, name string, args map[string]any) {
	if r == nil { return }
	r.mu.Lock(); defer r.mu.Unlock()
	r.pending = append(r.pending, atifToolCall{ToolCallID: id, FunctionName: name, Arguments: args})
}
func (r *atifRecorder) toolResult(id, result string) {
	if r == nil { return }
	r.mu.Lock(); defer r.mu.Unlock()
	r.results[id] = result
}

// agentMessage commits the actual final model text with all calls/results that
// occurred in its tool loop. Recorder failures only go to stderr.
func (r *atifRecorder) agentMessage(message string) {
	if r == nil || message == "" { return }
	r.mu.Lock(); defer r.mu.Unlock()
	step := atifStep{StepID: len(r.steps) + 1, Source: "agent", Message: message}
	if len(r.pending) > 0 {
		step.ToolCalls = append([]atifToolCall(nil), r.pending...)
		for _, call := range r.pending {
			if result, ok := r.results[call.ToolCallID]; ok {
				if step.Observation == nil { step.Observation = &atifObservation{} }
				step.Observation.Results = append(step.Observation.Results, atifToolResult{SourceCallID: call.ToolCallID, Content: result})
			}
		}
	}
	r.pending = nil
	r.results = make(map[string]string)
	r.steps = append(r.steps, step)
	r.writeLocked()
}

func (r *atifRecorder) writeLocked() {
	doc := atifDocument{SchemaVersion: "ATIF-v1.7", Agent: atifAgent{Name: AgentName, Version: "go"}, Steps: r.steps}
	data, err := json.Marshal(doc)
	if err != nil { fmt.Fprintf(os.Stderr, "relai ATIF marshal failed: %v\n", err); return }
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil { fmt.Fprintf(os.Stderr, "relai ATIF directory failed: %v\n", err); return }
	tmp, err := os.CreateTemp(filepath.Dir(r.path), ".trajectory-*")
	if err != nil { fmt.Fprintf(os.Stderr, "relai ATIF temp failed: %v\n", err); return }
	tmpName := tmp.Name()
	if _, err = tmp.Write(data); err == nil { err = tmp.Close() }
	if err == nil { err = os.Rename(tmpName, r.path) }
	if err != nil { _ = os.Remove(tmpName); fmt.Fprintf(os.Stderr, "relai ATIF write failed: %v\n", err) }
}
