package airline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// atifRecorder is enabled only for RELAI Harbor executions. Normal terminal
// sessions do not create a trajectory or alter their streaming behavior.
type atifRecorder struct {
	path string
	mu   sync.Mutex
}

type atifDocument struct {
	SchemaVersion string     `json:"schema_version"`
	Agent         atifAgent  `json:"agent"`
	Steps         []atifStep `json:"steps"`
}
type atifAgent struct { Name string `json:"name"`; Version string `json:"version"` }
type atifStep struct {
	StepID      int              `json:"step_id"`
	Source      string           `json:"source"`
	Message     string           `json:"message"`
	ToolCalls   []atifToolCall   `json:"tool_calls,omitempty"`
	Observation *atifObservation `json:"observation,omitempty"`
}
type atifToolCall struct { ToolCallID string `json:"tool_call_id"`; FunctionName string `json:"function_name"`; Arguments map[string]any `json:"arguments"` }
type atifObservation struct { Results []atifResult `json:"results"` }
type atifResult struct { SourceCallID string `json:"source_call_id"`; Content string `json:"content"` }

func newATIFRecorder() *atifRecorder {
	if path := os.Getenv("RELAI_ATIF_TRAJECTORY_PATH"); path != "" { return &atifRecorder{path: path} }
	return nil
}

func (r *atifRecorder) appendAgent(message string, calls []atifToolCall, results []atifResult) {
	if r == nil { return }
	r.mu.Lock(); defer r.mu.Unlock()
	doc := atifDocument{SchemaVersion: "ATIF-v1.7", Agent: atifAgent{Name: AgentName, Version: "go"}}
	if data, err := os.ReadFile(r.path); err == nil { _ = json.Unmarshal(data, &doc) }
	// A real provider tool-call message may contain no text. Preserve that fact
	// rather than manufacturing text; the call and its observed result remain linked.
	step := atifStep{StepID: len(doc.Steps) + 1, Source: "agent", Message: message, ToolCalls: calls}
	if len(results) != 0 { step.Observation = &atifObservation{Results: results} }
	doc.SchemaVersion = "ATIF-v1.7"
	doc.Agent = atifAgent{Name: AgentName, Version: "go"}
	doc.Steps = append(doc.Steps, step)
	data, err := json.Marshal(doc); if err != nil { return }
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil { return }
	tmp, err := os.CreateTemp(filepath.Dir(r.path), ".trajectory-*"); if err != nil { return }
	name := tmp.Name()
	if _, err = tmp.Write(data); err == nil { err = tmp.Close() } else { _ = tmp.Close() }
	if err == nil { _ = os.Rename(name, r.path) } else { _ = os.Remove(name) }
}
