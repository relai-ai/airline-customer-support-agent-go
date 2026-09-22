package airline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// atifRecorder records only data observed by the normal provider loop. It is
// enabled exclusively for RELAI Harbor runs and atomically replaces a complete
// cumulative snapshot after each observed turn.
type atifRecorder struct {
	path    string
	mu      sync.Mutex
	pending []atifToolCall
}

type atifDocument struct {
	SchemaVersion string     `json:"schema_version"`
	Agent         atifAgent  `json:"agent"`
	Steps         []atifStep `json:"steps"`
}

type atifAgent struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type atifStep struct {
	StepID      int              `json:"step_id"`
	Source      string           `json:"source"`
	Message     string           `json:"message"`
	ToolCalls   []atifToolCall   `json:"tool_calls,omitempty"`
	Observation *atifObservation `json:"observation,omitempty"`
}

type atifToolCall struct {
	ToolCallID   string         `json:"tool_call_id"`
	FunctionName string         `json:"function_name"`
	Arguments    map[string]any `json:"arguments"`
	Result       string         `json:"-"`
}

type atifObservation struct {
	Results []atifResult `json:"results"`
}

type atifResult struct {
	SourceCallID string `json:"source_call_id"`
	Content      string `json:"content"`
}

func newATIFRecorder() *atifRecorder {
	if path := os.Getenv("RELAI_ATIF_TRAJECTORY_PATH"); path != "" {
		return &atifRecorder{path: path}
	}
	return nil
}

func (r *atifRecorder) recordUser(message string) error {
	if r == nil || message == "" {
		return nil
	}
	return r.append(atifStep{Source: "user", Message: message})
}

func (r *atifRecorder) recordToolCall(id, name string, args map[string]any, result string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pending = append(r.pending, atifToolCall{ToolCallID: id, FunctionName: name, Arguments: args, Result: result})
}

// recordAssistant attaches pending real calls and results to the next actual,
// nonempty assistant text response. No placeholder message is manufactured for
// provider tool-use envelopes that contain no assistant text.
func (r *atifRecorder) recordAssistant(message string) error {
	if r == nil || message == "" {
		return nil
	}
	r.mu.Lock()
	calls := r.pending
	r.pending = nil
	r.mu.Unlock()
	step := atifStep{Source: "agent", Message: message}
	if len(calls) > 0 {
		step.ToolCalls = make([]atifToolCall, len(calls))
		results := make([]atifResult, len(calls))
		for i, call := range calls {
			step.ToolCalls[i] = atifToolCall{ToolCallID: call.ToolCallID, FunctionName: call.FunctionName, Arguments: call.Arguments}
			results[i] = atifResult{SourceCallID: call.ToolCallID, Content: call.Result}
		}
		step.Observation = &atifObservation{Results: results}
	}
	return r.append(step)
}

func (r *atifRecorder) append(step atifStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	doc := atifDocument{SchemaVersion: "ATIF-v1.7", Agent: atifAgent{Name: AgentName, Version: "go"}}
	data, err := os.ReadFile(r.path)
	if err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("read existing ATIF trajectory: %w", err)
		}
	}
	doc.SchemaVersion = "ATIF-v1.7"
	doc.Agent = atifAgent{Name: AgentName, Version: "go"}
	step.StepID = len(doc.Steps) + 1
	doc.Steps = append(doc.Steps, step)
	encoded, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(r.path), ".trajectory-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(encoded); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, r.path)
}
