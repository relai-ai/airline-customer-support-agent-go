package airline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// ATIFRecorder records only events observed by the existing agent loop. It is
// inactive in ordinary application runs, where RELAI_ATIF_TRAJECTORY_PATH is unset.
type ATIFRecorder struct {
	mu   sync.Mutex
	path string
	doc  atifDocument
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
	StepID      int               `json:"step_id"`
	Source      string            `json:"source"`
	Message     string            `json:"message"`
	ToolCalls   []atifToolCall    `json:"tool_calls,omitempty"`
	Observation *atifObservation  `json:"observation,omitempty"`
}

type atifToolCall struct {
	ToolCallID  string         `json:"tool_call_id"`
	FunctionName string        `json:"function_name"`
	Arguments   map[string]any `json:"arguments"`
}

type atifObservation struct {
	Results []atifToolResult `json:"results"`
}

type atifToolResult struct {
	SourceCallID string `json:"source_call_id"`
	Content      string `json:"content"`
}

var activeATIF *ATIFRecorder

// BeginATIFTurn enables opt-in tracing for one adapter invocation. Existing
// valid steps are retained so sequential Harbor turns produce one cumulative
// document. It records the actual final user input when present.
func BeginATIFTurn(history []Message) {
	path := os.Getenv("RELAI_ATIF_TRAJECTORY_PATH")
	if path == "" {
		activeATIF = nil
		return
	}
	recorder := &ATIFRecorder{path: path, doc: atifDocument{
		SchemaVersion: "ATIF-v1.7",
		Agent: atifAgent{Name: AgentName, Version: "go"},
	}}
	if bytes, err := os.ReadFile(path); err == nil {
		var previous atifDocument
		if json.Unmarshal(bytes, &previous) == nil && previous.SchemaVersion == "ATIF-v1.7" && previous.Agent.Name != "" && previous.Agent.Version != "" {
			recorder.doc = previous
		}
	}
	activeATIF = recorder
	if len(history) > 0 {
		last := history[len(history)-1]
		if last.Role == "user" && last.Content != "" {
			recorder.append(atifStep{Source: "user", Message: last.Content})
		}
	}
}

// RecordATIFModelEvent is called at the actual provider model/tool boundary.
// A tool-bearing provider message is saved only when the provider supplied its
// required nonempty assistant message; no synthetic text is introduced.
func RecordATIFModelEvent(message string, calls []atifToolCall, results []atifToolResult) {
	if activeATIF == nil || message == "" {
		return
	}
	step := atifStep{Source: "agent", Message: message, ToolCalls: calls}
	if len(results) > 0 {
		step.Observation = &atifObservation{Results: results}
	}
	activeATIF.append(step)
}

func (r *ATIFRecorder) append(step atifStep) {
	r.mu.Lock()
	defer r.mu.Unlock()
	step.StepID = len(r.doc.Steps) + 1
	r.doc.Steps = append(r.doc.Steps, step)
	// Keep the previous valid snapshot if marshaling or replacement fails.
	bytes, err := json.Marshal(r.doc)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return
	}
	temp, err := os.CreateTemp(filepath.Dir(r.path), ".trajectory-*.json")
	if err != nil {
		return
	}
	name := temp.Name()
	if _, err = temp.Write(bytes); err == nil {
		err = temp.Close()
	} else {
		_ = temp.Close()
	}
	if err == nil {
		err = os.Rename(name, r.path)
	}
	if err != nil {
		_ = os.Remove(name)
	}
}
