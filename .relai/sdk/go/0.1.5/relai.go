package relai

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// LearningEnvironmentSchemaVersion is the current learning environment wire schema.
	LearningEnvironmentSchemaVersion = "relai.learning_environment.v1"
	// BenchmarkSchemaVersion is the current benchmark wire schema.
	BenchmarkSchemaVersion = "relai.benchmark.v1"
	// SimulationResultSchemaVersion is the current simulation result wire schema.
	SimulationResultSchemaVersion = "relai.simulation_result.v1"
)

// JSONValue is any JSON-serializable Go value.
type JSONValue = any

// ModelSpec identifies a model by provider and name.
type ModelSpec struct {
	Name     string `json:"name"`
	Provider string `json:"provider,omitempty"`
}

// Input is a RELAI learning environment input strategy.
type Input interface {
	Type() string
	inputPayload() any
}

// FixedTurn is one fixed user turn passed to an agent.
type FixedTurn struct {
	Content JSONValue `json:"content"`
}

// FixedInput replays exact user turns.
type FixedInput struct {
	Turns []FixedTurn
}

// FixedInputPayload is the JSON payload produced by FixedInput.
type FixedInputPayload struct {
	Type  string      `json:"type"`
	Turns []FixedTurn `json:"turns"`
}

// Type returns the RELAI input type.
func (FixedInput) Type() string { return "fixed" }

// Serialize returns the JSON payload for the fixed input.
func (input FixedInput) Serialize() FixedInputPayload {
	return FixedInputPayload{Type: input.Type(), Turns: input.Turns}
}

func (input FixedInput) inputPayload() any { return input.Serialize() }

// PersonaInput asks the simulator to generate a conversation from a persona.
type PersonaInput struct {
	Persona     string
	Intent      string
	MaxTurns    int
	Model       ModelSpec
	SeedMessage JSONValue
}

// PersonaInputPayload is the JSON payload produced by PersonaInput.
type PersonaInputPayload struct {
	Type        string    `json:"type"`
	Persona     string    `json:"persona"`
	Intent      string    `json:"intent,omitempty"`
	MaxTurns    int       `json:"max_turns"`
	Model       ModelSpec `json:"model"`
	SeedMessage JSONValue `json:"seed_message,omitempty"`
}

// Type returns the RELAI input type.
func (PersonaInput) Type() string { return "persona" }

// Serialize returns the JSON payload for the persona input.
func (input PersonaInput) Serialize() PersonaInputPayload {
	return PersonaInputPayload{
		Type:        input.Type(),
		Persona:     input.Persona,
		Intent:      input.Intent,
		MaxTurns:    input.MaxTurns,
		Model:       input.Model,
		SeedMessage: input.SeedMessage,
	}
}

func (input PersonaInput) inputPayload() any { return input.Serialize() }

// FixedComponentInput calls a component with fixed positional and named arguments.
type FixedComponentInput struct {
	Args      []JSONValue
	NamedArgs map[string]JSONValue
}

// FixedComponentInputPayload is the JSON payload produced by FixedComponentInput.
type FixedComponentInputPayload struct {
	Type      string               `json:"type"`
	Args      []JSONValue          `json:"args"`
	NamedArgs map[string]JSONValue `json:"kwargs"`
}

// Type returns the RELAI input type.
func (FixedComponentInput) Type() string { return "component_fixed" }

// Serialize returns the JSON payload for the fixed component input.
func (input FixedComponentInput) Serialize() FixedComponentInputPayload {
	return FixedComponentInputPayload{Type: input.Type(), Args: input.Args, NamedArgs: input.NamedArgs}
}

func (input FixedComponentInput) inputPayload() any { return input.Serialize() }

// GeneratedComponentInput asks RELAI to generate component input values.
type GeneratedComponentInput struct {
	Instructions string
	Model        ModelSpec
	InputSchema  JSONValue
}

// GeneratedComponentInputPayload is the JSON payload produced by GeneratedComponentInput.
type GeneratedComponentInputPayload struct {
	Type         string    `json:"type"`
	Instructions string    `json:"instructions"`
	Model        ModelSpec `json:"model"`
	InputSchema  JSONValue `json:"input_schema,omitempty"`
}

// Type returns the RELAI input type.
func (GeneratedComponentInput) Type() string { return "component_generated" }

// Serialize returns the JSON payload for the generated component input.
func (input GeneratedComponentInput) Serialize() GeneratedComponentInputPayload {
	return GeneratedComponentInputPayload{
		Type:         input.Type(),
		Instructions: input.Instructions,
		Model:        input.Model,
		InputSchema:  input.InputSchema,
	}
}

func (input GeneratedComponentInput) inputPayload() any { return input.Serialize() }

// Target describes what a learning environment exercises.
type Target interface {
	Type() string
	targetPayload() TargetPayload
}

// TargetPayload is the JSON payload produced by a target.
type TargetPayload struct {
	Type        string `json:"type"`
	AgentTarget string `json:"agent_target,omitempty"`
	ImportPath  string `json:"import_path,omitempty"`
}

// AgentTarget runs a full agent through the simulator adapter.
type AgentTarget struct {
	// AgentTarget selects one logical agent when a project uses a shared runtime.
	// An empty value preserves the project's default agent behavior.
	AgentTarget string
}

// Type returns the RELAI target type.
func (AgentTarget) Type() string { return "agent" }

// Serialize returns the JSON payload for an agent target.
func (target AgentTarget) Serialize() TargetPayload {
	return target.targetPayload()
}

func (target AgentTarget) targetPayload() TargetPayload {
	return TargetPayload{Type: target.Type(), AgentTarget: target.AgentTarget}
}

// ComponentTarget runs one component function or method.
type ComponentTarget struct {
	ImportPath string
}

// Type returns the RELAI target type.
func (ComponentTarget) Type() string { return "component" }

// Serialize returns the JSON payload for a component target.
func (target ComponentTarget) Serialize() TargetPayload {
	return target.targetPayload()
}

func (target ComponentTarget) targetPayload() TargetPayload {
	return TargetPayload{Type: target.Type(), ImportPath: target.ImportPath}
}

// EvaluationResult is the score and feedback returned by a code evaluator.
type EvaluationResult struct {
	Score    float64 `json:"score"`
	Feedback string  `json:"feedback"`
}

// Evaluator scores a simulation result.
type Evaluator interface {
	ID() string
	Type() string
	evaluatorPayload() EvaluatorPayload
	Evaluate(SimulationResult) (EvaluationResult, error)
}

// EvaluatorPayload is the JSON payload produced by an evaluator.
type EvaluatorPayload struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Description  string     `json:"description"`
	Instructions string     `json:"instructions,omitempty"`
	MinScore     float64    `json:"min_score"`
	MaxScore     float64    `json:"max_score"`
	Model        *ModelSpec `json:"model,omitempty"`
	Scope        string     `json:"scope,omitempty"`
	AgentTarget  string     `json:"agent_target,omitempty"`
}

// CodeEvaluator scores simulation results using a Go function.
type CodeEvaluator struct {
	EvaluatorID string
	Description string
	MinScore    float64
	MaxScore    float64
	Scope       string
	AgentTarget string
	Fn          func(SimulationResult) (EvaluationResult, error)
}

// ID returns the evaluator id.
func (e CodeEvaluator) ID() string { return e.EvaluatorID }

// Type returns the RELAI evaluator type.
func (e CodeEvaluator) Type() string { return "code" }

// Serialize returns the JSON payload for a code evaluator.
func (e CodeEvaluator) Serialize() EvaluatorPayload {
	return e.evaluatorPayload()
}

func (e CodeEvaluator) evaluatorPayload() EvaluatorPayload {
	minScore, maxScore := defaultScoreBounds(e.MinScore, e.MaxScore)
	return EvaluatorPayload{
		ID:          e.EvaluatorID,
		Type:        e.Type(),
		Description: e.Description,
		MinScore:    minScore,
		MaxScore:    maxScore,
		Scope:       e.Scope,
		AgentTarget: e.AgentTarget,
	}
}

// Evaluate runs the evaluator function.
func (e CodeEvaluator) Evaluate(result SimulationResult) (EvaluationResult, error) {
	minScore, maxScore, err := resolveScoreBounds(e.MinScore, e.MaxScore)
	if err != nil {
		return EvaluationResult{}, fmt.Errorf("code evaluator %q has invalid score bounds: %w", e.ID(), err)
	}
	if e.Fn == nil {
		return EvaluationResult{}, errors.New("code evaluator requires Fn")
	}
	evaluation, err := e.Fn(result)
	if err != nil {
		return EvaluationResult{}, err
	}
	if err := validateEvaluationResult(e.ID(), evaluation, minScore, maxScore); err != nil {
		return EvaluationResult{}, err
	}
	return evaluation, nil
}

// LLMJudgeEvaluator asks RELAI to score simulation results with an LLM judge.
type LLMJudgeEvaluator struct {
	EvaluatorID  string
	Description  string
	Instructions string
	MinScore     float64
	MaxScore     float64
	Model        ModelSpec
	Scope        string
	AgentTarget  string
}

// ID returns the evaluator id.
func (e LLMJudgeEvaluator) ID() string { return e.EvaluatorID }

// Type returns the RELAI evaluator type.
func (e LLMJudgeEvaluator) Type() string { return "llm_judge" }

// Serialize returns the JSON payload for an LLM judge evaluator.
func (e LLMJudgeEvaluator) Serialize() EvaluatorPayload {
	return e.evaluatorPayload()
}

func (e LLMJudgeEvaluator) evaluatorPayload() EvaluatorPayload {
	minScore, maxScore := defaultScoreBounds(e.MinScore, e.MaxScore)
	return EvaluatorPayload{
		ID:           e.EvaluatorID,
		Type:         e.Type(),
		Description:  e.Description,
		Instructions: e.Instructions,
		MinScore:     minScore,
		MaxScore:     maxScore,
		Model:        &e.Model,
		Scope:        e.Scope,
		AgentTarget:  e.AgentTarget,
	}
}

// Evaluate scores a simulation result with an LLM judge using a background context.
// Prefer EvaluateContext when the caller has a context to propagate.
func (e LLMJudgeEvaluator) Evaluate(result SimulationResult) (EvaluationResult, error) {
	return e.EvaluateContext(context.Background(), result)
}

// EvaluateContext scores a simulation result with an ADK Go LLM judge.
func (e LLMJudgeEvaluator) EvaluateContext(ctx context.Context, result SimulationResult) (EvaluationResult, error) {
	return runLLMJudge(ctx, e, result, defaultJudgeModel)
}

// BenchmarkSample is one CSV row supplied to a benchmark environment builder.
type BenchmarkSample struct {
	Index int
	ID    string
	Row   map[string]string
}

// Benchmark defines a RELAI benchmark.
type Benchmark struct {
	ID               string
	Name             string
	Description      string
	RequiredColumns  []string
	SampleIDColumn   string
	AgentTarget      string
	DatasetRef       map[string]any
	BuildEnvironment func(BenchmarkSample) (Environment, error)
}

// BenchmarkPayload is the JSON payload produced by Benchmark.
type BenchmarkPayload struct {
	SchemaVersion   string         `json:"schema_version"`
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	RequiredColumns []string       `json:"required_columns"`
	SampleIDColumn  string         `json:"sample_id_column,omitempty"`
	AgentTarget     string         `json:"agent_target,omitempty"`
	DatasetRef      map[string]any `json:"dataset_ref,omitempty"`
}

// Serialize returns the JSON payload for a benchmark.
func (b Benchmark) Serialize() BenchmarkPayload {
	return BenchmarkPayload{
		SchemaVersion:   BenchmarkSchemaVersion,
		ID:              b.ID,
		Name:            b.Name,
		Description:     b.Description,
		RequiredColumns: b.RequiredColumns,
		SampleIDColumn:  b.SampleIDColumn,
		AgentTarget:     b.AgentTarget,
		DatasetRef:      b.DatasetRef,
	}
}

// StoredBenchmarkCSV returns a stored CSV dataset reference.
func StoredBenchmarkCSV(id string) map[string]any {
	return map[string]any{"type": "stored_csv", "id": id}
}

// LocalBenchmarkCSV returns a local CSV dataset reference.
func LocalBenchmarkCSV(path string) map[string]any {
	return map[string]any{"type": "local_csv", "path": path}
}

// Environment defines a RELAI learning environment.
type Environment struct {
	ID          string
	Name        string
	Description string
	Tags        []string
	Target      Target
	Input       Input
	Mocks       map[string]any
	Evaluators  []Evaluator
}

// EnvironmentPayload is the JSON payload produced by Environment.
type EnvironmentPayload struct {
	SchemaVersion string             `json:"schema_version"`
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Description   string             `json:"description"`
	Tags          []string           `json:"tags"`
	Target        TargetPayload      `json:"target"`
	Input         any                `json:"input,omitempty"`
	Mocks         map[string]any     `json:"mocks"`
	Evaluators    []EvaluatorPayload `json:"evaluators"`
}

// Serialize returns the JSON payload for a learning environment.
func (e Environment) Serialize() EnvironmentPayload {
	target := e.Target
	if target == nil {
		target = AgentTarget{}
	}
	payload := EnvironmentPayload{
		SchemaVersion: LearningEnvironmentSchemaVersion,
		ID:            e.ID,
		Name:          e.Name,
		Description:   e.Description,
		Tags:          e.Tags,
		Target:        target.targetPayload(),
		Mocks:         nonNilMap(e.Mocks),
		Evaluators:    serializeEvaluators(e.Evaluators),
	}
	if e.Input != nil {
		payload.Input = e.Input.inputPayload()
	}
	return payload
}

// AgentTurnResult is the response from one simulator adapter turn.
type AgentTurnResult struct {
	AssistantMessage JSONValue          `json:"assistant_message"`
	Metadata         map[string]any     `json:"metadata,omitempty"`
	ToolCalls        []ToolCallRecord   `json:"tool_calls,omitempty"`
	ToolResults      []ToolResultRecord `json:"tool_results,omitempty"`
}

// ToolCallRecord describes a tool call made during a turn.
type ToolCallRecord struct {
	Name      string         `json:"name"`
	Arguments JSONValue      `json:"arguments,omitempty"`
	CallID    string         `json:"call_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ToolResultRecord describes a tool result observed during a turn.
type ToolResultRecord struct {
	Name     string         `json:"name"`
	Result   JSONValue      `json:"result,omitempty"`
	Error    string         `json:"error,omitempty"`
	CallID   string         `json:"call_id,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// AgentAdapter connects the RELAI runner to project agent code.
type AgentAdapter interface {
	RunTurn(context.Context, JSONValue) (AgentTurnResult, error)
}

// AdapterCapability is a behavior implemented and verified by a project adapter.
type AdapterCapability string

const (
	AdapterCapabilityRunTurn       AdapterCapability = "run_turn"
	AdapterCapabilityNamedTargets  AdapterCapability = "named_targets"
	AdapterCapabilityToolOverrides AdapterCapability = "tool_overrides"
	AdapterCapabilityToolEvents    AdapterCapability = "tool_events"
	AdapterCapabilityRunComponent  AdapterCapability = "run_component"
)

var adapterCapabilities = map[AdapterCapability]struct{}{
	AdapterCapabilityRunTurn: {}, AdapterCapabilityNamedTargets: {},
	AdapterCapabilityToolOverrides: {}, AdapterCapabilityToolEvents: {},
	AdapterCapabilityRunComponent: {},
}

// AdapterRuntime contains RELAI services consumed by project-owned integration code.
// ToolOverrides are deliberately opaque: the generated adapter owns conversion to
// the project's concrete function signatures.
type AdapterRuntime struct {
	ToolOverrides map[string]any
	MockCalls     []MockCallRecord
}

// MockCallRecord records one adapter-owned named tool override invocation.
type MockCallRecord struct {
	Target    string
	Arguments JSONValue
	Result    JSONValue
	Error     string
}

// NewAdapterRuntime creates a runtime for one learning environment.
func NewAdapterRuntime(overrides map[string]any) *AdapterRuntime {
	toolOverrides := map[string]any{}
	for target, replacement := range overrides {
		if !strings.Contains(target, ":") {
			toolOverrides[target] = replacement
		}
	}
	return &AdapterRuntime{ToolOverrides: toolOverrides, MockCalls: []MockCallRecord{}}
}

// HasToolOverride reports whether a named override is configured.
func (runtime *AdapterRuntime) HasToolOverride(name string) bool {
	_, ok := runtime.ToolOverrides[name]
	return ok
}

// RecordMockCall appends an invocation translated by the project adapter.
func (runtime *AdapterRuntime) RecordMockCall(call MockCallRecord) {
	runtime.MockCalls = append(runtime.MockCalls, call)
}

// CapabilityAdapter declares the behaviors implemented by a project adapter.
type CapabilityAdapter interface {
	AdapterCapabilities() []AdapterCapability
}

// AdapterSelfChecker behaviorally validates optional adapter capabilities.
type AdapterSelfChecker interface {
	SelfCheck(context.Context, *AdapterRuntime) (map[AdapterCapability]bool, error)
}

// ValidateAdapterCapabilities validates the stable RELAI adapter contract.
func ValidateAdapterCapabilities(adapter any) (map[AdapterCapability]struct{}, error) {
	capable, ok := adapter.(CapabilityAdapter)
	if !ok {
		return nil, errors.New("simulator adapter must implement AdapterCapabilities")
	}
	declared := map[AdapterCapability]struct{}{}
	for _, capability := range capable.AdapterCapabilities() {
		if _, ok := adapterCapabilities[capability]; !ok {
			return nil, fmt.Errorf("simulator adapter declares unknown capability %q", capability)
		}
		declared[capability] = struct{}{}
	}
	if _, ok := declared[AdapterCapabilityRunTurn]; !ok {
		return nil, errors.New("simulator adapter must declare the run_turn capability")
	}
	if _, ok := adapter.(AgentAdapter); !ok {
		return nil, errors.New("simulator adapter must implement AgentAdapter")
	}
	if _, declaredTarget := declared[AdapterCapabilityNamedTargets]; declaredTarget {
		if _, ok := adapter.(AgentTargetConfigurer); !ok {
			return nil, errors.New("named_targets capability requires AgentTargetConfigurer")
		}
	}
	if _, declaredComponent := declared[AdapterCapabilityRunComponent]; declaredComponent {
		if _, ok := adapter.(ComponentAdapter); !ok {
			return nil, errors.New("run_component capability requires ComponentAdapter")
		}
	}
	return declared, nil
}

// ValidateAdapterSelfCheck requires every optional capability to pass its check.
func ValidateAdapterSelfCheck(
	capabilities map[AdapterCapability]struct{},
	results map[AdapterCapability]bool,
) error {
	for capability := range capabilities {
		if capability != AdapterCapabilityRunTurn && !results[capability] {
			return fmt.Errorf("adapter self-check did not verify capability %q", capability)
		}
	}
	return nil
}

// AgentTargetConfigurer is optionally implemented by adapters that support
// named logical agents behind one shared runtime. The runner calls it before
// the first turn when AgentTarget.AgentTarget is set.
type AgentTargetConfigurer interface {
	ConfigureAgentTarget(context.Context, AgentTarget) error
}

// ComponentAdapter connects the RELAI runner to project component code.
// Implement it on the adapter passed to RunEnvironment when executing a
// ComponentTarget.
type ComponentAdapter interface {
	RunComponent(context.Context, ComponentTarget, FixedComponentInput) (JSONValue, error)
}

// TranscriptEvent is one JSONL event in a simulator transcript.
type TranscriptEvent struct {
	EventType string         `json:"event_type"`
	TurnIndex int            `json:"turn_index"`
	Timestamp string         `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// ComponentRunTiming records the execution time of one component invocation.
type ComponentRunTiming struct {
	Target     string `json:"target"`
	DurationMS int64  `json:"duration_ms"`
}

// SimulationTimings records execution time in milliseconds. It excludes input
// generation and evaluator execution.
type SimulationTimings struct {
	TotalDurationMS int64                `json:"total_duration_ms"`
	ComponentRuns   []ComponentRunTiming `json:"component_runs"`
}

// SimulationResult is the result emitted by a simulator run.
type SimulationResult struct {
	SchemaVersion         string             `json:"schema_version"`
	LearningEnvironmentID string             `json:"learning_environment_id"`
	RunID                 string             `json:"run_id"`
	TranscriptPath        string             `json:"transcript_path"`
	EventsCount           int                `json:"events_count"`
	FinalOutput           JSONValue          `json:"final_output,omitempty"`
	StopReason            string             `json:"stop_reason"`
	Evaluations           []EvaluationEntry  `json:"evaluations"`
	EvaluationErrors      []EvaluationError  `json:"evaluation_errors"`
	Timings               *SimulationTimings `json:"timings,omitempty"`
	Metadata              map[string]any     `json:"metadata,omitempty"`
	Events                []TranscriptEvent  `json:"-"`
}

// EvaluationEntry is one successful evaluator result.
type EvaluationEntry struct {
	EvaluatorID       string  `json:"evaluator_id"`
	Score             float64 `json:"score"`
	Feedback          string  `json:"feedback"`
	MinScore          float64 `json:"min_score"`
	MaxScore          float64 `json:"max_score"`
	IsGlobalEvaluator bool    `json:"is_global_evaluator,omitempty"`
}

// EvaluationError is one evaluator failure.
type EvaluationError struct {
	EvaluatorID       string `json:"evaluator_id"`
	Message           string `json:"message"`
	IsGlobalEvaluator bool   `json:"is_global_evaluator,omitempty"`
}

type globalEvaluator struct {
	evaluator Evaluator
}

func (e globalEvaluator) ID() string { return e.evaluator.ID() }

func (e globalEvaluator) Type() string { return e.evaluator.Type() }

func (e globalEvaluator) evaluatorPayload() EvaluatorPayload { return e.evaluator.evaluatorPayload() }

func (e globalEvaluator) Evaluate(result SimulationResult) (EvaluationResult, error) {
	return e.evaluator.Evaluate(result)
}

// CombineEvaluators combines environment and global evaluators, marks the global
// entries in the simulation result, and rejects duplicate evaluator IDs.
func CombineEvaluators(environmentEvaluators, globalEvaluators []Evaluator) ([]Evaluator, error) {
	combined := make([]Evaluator, 0, len(environmentEvaluators)+len(globalEvaluators))
	seen := map[string]struct{}{}
	for _, evaluator := range environmentEvaluators {
		if _, exists := seen[evaluator.ID()]; exists {
			return nil, fmt.Errorf("evaluator id %q is duplicated", evaluator.ID())
		}
		seen[evaluator.ID()] = struct{}{}
		combined = append(combined, evaluator)
	}
	for _, evaluator := range globalEvaluators {
		if _, exists := seen[evaluator.ID()]; exists {
			return nil, fmt.Errorf("evaluator id %q is duplicated", evaluator.ID())
		}
		seen[evaluator.ID()] = struct{}{}
		combined = append(combined, globalEvaluator{evaluator: evaluator})
	}
	return combined, nil
}

// FilterGlobalEvaluatorsForEnvironment returns the global evaluators whose scope
// matches the environment. Global evaluator scopes must be `end-to-end` or a
// component ID declared in .relai/learning-env-context.json.
func FilterGlobalEvaluatorsForEnvironment(projectRoot string, evaluators []Evaluator, environment Environment) ([]Evaluator, error) {
	if len(evaluators) == 0 {
		return nil, nil
	}
	componentIDs, componentIDsByImportPath, err := loadLearningEnvironmentComponents(projectRoot)
	if err != nil {
		return nil, err
	}
	for _, evaluator := range evaluators {
		if err := validateGlobalEvaluatorScope(evaluator, componentIDs); err != nil {
			return nil, err
		}
	}
	scope, err := deriveEnvironmentScope(environment, componentIDsByImportPath)
	if err != nil {
		return nil, err
	}
	filtered := make([]Evaluator, 0, len(evaluators))
	environmentTarget := ""
	if target, ok := agentTarget(environment.Target); ok {
		environmentTarget = target.AgentTarget
	}
	for _, evaluator := range evaluators {
		payload := evaluator.evaluatorPayload()
		if payload.Scope == scope && (payload.AgentTarget == "" || payload.AgentTarget == environmentTarget) {
			filtered = append(filtered, evaluator)
		}
	}
	return filtered, nil
}

// ValidateGlobalEvaluator verifies that an evaluator has a supported global
// scope for the project. Call it when loading a global evaluator source file.
func ValidateGlobalEvaluator(projectRoot string, evaluator Evaluator) error {
	if _, _, err := evaluatorScoreBounds(evaluator); err != nil {
		return err
	}
	componentIDs, _, err := loadLearningEnvironmentComponents(projectRoot)
	if err != nil {
		return err
	}
	return validateGlobalEvaluatorScope(evaluator, componentIDs)
}

func validateGlobalEvaluatorScope(evaluator Evaluator, componentIDs map[string]struct{}) error {
	scope := strings.TrimSpace(evaluator.evaluatorPayload().Scope)
	if scope == "" {
		return fmt.Errorf("global evaluator %q must declare a non-empty scope", evaluator.ID())
	}
	if scope == "end-to-end" {
		return nil
	}
	if _, exists := componentIDs[scope]; exists {
		return nil
	}
	return fmt.Errorf("global evaluator %q scope must be `end-to-end` or a component id from .relai/learning-env-context.json; got %q", evaluator.ID(), scope)
}

func deriveEnvironmentScope(environment Environment, componentIDsByImportPath map[string]string) (string, error) {
	if _, isComponent := componentTarget(environment.Target); !isComponent {
		return "end-to-end", nil
	}
	target, _ := componentTarget(environment.Target)
	if componentID := componentIDsByImportPath[target.ImportPath]; componentID != "" {
		return componentID, nil
	}
	candidates := make([]string, 0, len(environment.Tags))
	for _, tag := range environment.Tags {
		if tag != "" && !isBenchmarkSampleTag(tag) {
			candidates = append(candidates, tag)
		}
	}
	if containsString(candidates, "end-to-end") {
		return "", errors.New("component learning environment cannot use the `end-to-end` scope tag")
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	return "", fmt.Errorf("could not derive component scope for target %q; ensure learning-env-context.json includes the component or the environment tags expose exactly one scope tag", target.ImportPath)
}

func isBenchmarkSampleTag(tag string) bool {
	if !strings.HasPrefix(tag, "benchmark-") {
		return false
	}
	benchmarkAndSample := strings.TrimPrefix(tag, "benchmark-")
	marker := "-sample-"
	markerIndex := strings.LastIndex(benchmarkAndSample, marker)
	if markerIndex <= 0 || markerIndex+len(marker) == len(benchmarkAndSample) {
		return false
	}
	sampleIndex := benchmarkAndSample[markerIndex+len(marker):]
	_, err := strconv.Atoi(sampleIndex)
	return err == nil
}

type learningEnvironmentContext struct {
	Components []struct {
		ID         string `json:"id"`
		ImportPath string `json:"import_path"`
	} `json:"components"`
}

func loadLearningEnvironmentComponents(projectRoot string) (map[string]struct{}, map[string]string, error) {
	path := filepath.Join(projectRoot, ".relai", "learning-env-context.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]struct{}{}, map[string]string{}, nil
		}
		return nil, nil, err
	}
	var context learningEnvironmentContext
	if err := json.Unmarshal(contents, &context); err != nil {
		return nil, nil, fmt.Errorf("read learning environment component context: %w", err)
	}
	ids := map[string]struct{}{}
	idsByImportPath := map[string]string{}
	for _, component := range context.Components {
		if component.ID != "" {
			ids[component.ID] = struct{}{}
		}
		if component.ID != "" && component.ImportPath != "" {
			idsByImportPath[component.ImportPath] = component.ID
		}
	}
	return ids, idsByImportPath, nil
}

// RunEnvironment runs a learning environment through an adapter and writes a transcript.
// Operational failures may return both a partial result and a non-nil error.
func RunEnvironment(ctx context.Context, projectRoot string, environment Environment, adapter AgentAdapter) (SimulationResult, error) {
	return RunEnvironmentWithRuntime(ctx, projectRoot, environment, adapter, NewAdapterRuntime(environment.Mocks))
}

// RunEnvironmentWithRuntime runs an agent environment with the runtime used to
// construct its project adapter.
func RunEnvironmentWithRuntime(ctx context.Context, projectRoot string, environment Environment, adapter AgentAdapter, runtime *AdapterRuntime) (SimulationResult, error) {
	return runAgentEnvironmentWithModelFactory(ctx, projectRoot, environment, adapter, runtime, defaultJudgeModel)
}

// RunComponentEnvironment runs a component learning environment through a component adapter.
// Component invocation failures are captured in the result; setup and cancellation
// failures may return both a partial result and a non-nil error.
func RunComponentEnvironment(ctx context.Context, projectRoot string, environment Environment, adapter ComponentAdapter) (SimulationResult, error) {
	return RunComponentEnvironmentWithRuntime(ctx, projectRoot, environment, adapter, NewAdapterRuntime(environment.Mocks))
}

// RunComponentEnvironmentWithRuntime runs a component environment with the
// runtime used to construct its project adapter.
func RunComponentEnvironmentWithRuntime(ctx context.Context, projectRoot string, environment Environment, adapter ComponentAdapter, runtime *AdapterRuntime) (SimulationResult, error) {
	return runComponentEnvironmentWithModelFactory(ctx, projectRoot, environment, adapter, runtime, defaultJudgeModel)
}

type transcriptRecorder func(string, int, map[string]any)

type environmentExecution struct {
	finalOutput JSONValue
	stopReason  string
	timings     SimulationTimings
	metadata    map[string]any
}

func runAgentEnvironmentWithModelFactory(ctx context.Context, projectRoot string, environment Environment, adapter AgentAdapter, runtime *AdapterRuntime, factory judgeModelFactory) (SimulationResult, error) {
	if _, ok := componentTarget(environment.Target); ok {
		return SimulationResult{}, errors.New("RunEnvironment requires an agent target; use RunComponentEnvironment for a component target")
	}
	return executeEnvironment(ctx, projectRoot, environment, func(record transcriptRecorder) (environmentExecution, error) {
		if target, ok := agentTarget(environment.Target); ok && target.AgentTarget != "" {
			configurer, ok := adapter.(AgentTargetConfigurer)
			if !ok {
				return environmentExecution{stopReason: "agent error"}, fmt.Errorf("agent target %q requires the adapter to implement relai.AgentTargetConfigurer", target.AgentTarget)
			}
			if err := configurer.ConfigureAgentTarget(ctx, target); err != nil {
				return environmentExecution{stopReason: "agent error"}, fmt.Errorf("configure agent target %q: %w", target.AgentTarget, err)
			}
		}
		finalOutput, stopReason, timings, err := runAgentInput(ctx, environment.Input, adapter, runtime, factory, record)
		return environmentExecution{finalOutput: finalOutput, stopReason: stopReason, timings: timings}, err
	})
}

func runComponentEnvironmentWithModelFactory(ctx context.Context, projectRoot string, environment Environment, adapter ComponentAdapter, runtime *AdapterRuntime, factory judgeModelFactory) (SimulationResult, error) {
	target, ok := componentTarget(environment.Target)
	if !ok {
		return SimulationResult{}, errors.New("RunComponentEnvironment requires a component target")
	}
	return executeEnvironment(ctx, projectRoot, environment, func(record transcriptRecorder) (environmentExecution, error) {
		if err := ctx.Err(); err != nil {
			return environmentExecution{stopReason: "component error"}, err
		}
		componentInput, generated, err := resolveComponentInput(ctx, target, environment.Input, factory)
		if err != nil {
			return environmentExecution{stopReason: "component error"}, err
		}
		if generated {
			record("component_input_generated", 0, map[string]any{"target": target.ImportPath, "input": componentInput.Serialize()})
		}
		record("component_call", 0, map[string]any{"target": target.ImportPath, "input": componentInput.Serialize()})
		recordedRuntimeCalls := runtimeMockCallCount(runtime)
		startedAt := time.Now()
		output, err := adapter.RunComponent(ctx, target, componentInput)
		recordRuntimeMockCalls(record, runtime, recordedRuntimeCalls, 0)
		timings := componentTimings(target.ImportPath, elapsedMilliseconds(startedAt))
		if err != nil {
			errorData := map[string]any{"message": err.Error(), "error_type": "Error"}
			record("component_error", 0, map[string]any{"target": target.ImportPath, "message": err.Error(), "error_type": "Error"})
			if isContextError(err) || ctx.Err() != nil {
				return environmentExecution{stopReason: "component error", timings: timings, metadata: map[string]any{"component_error": errorData}}, err
			}
			return environmentExecution{stopReason: "component error", timings: timings, metadata: map[string]any{"component_error": errorData}}, nil
		}
		if err := ctx.Err(); err != nil {
			errorData := map[string]any{"message": err.Error(), "error_type": "Error"}
			record("component_error", 0, map[string]any{"target": target.ImportPath, "message": err.Error(), "error_type": "Error"})
			return environmentExecution{stopReason: "component error", timings: timings, metadata: map[string]any{"component_error": errorData}}, err
		}
		record("component_result", 0, map[string]any{"target": target.ImportPath, "result": output})
		return environmentExecution{finalOutput: output, stopReason: "component complete", timings: timings}, nil
	})
}

func runtimeMockCallCount(runtime *AdapterRuntime) int {
	if runtime == nil {
		return 0
	}
	return len(runtime.MockCalls)
}

func recordRuntimeMockCalls(record transcriptRecorder, runtime *AdapterRuntime, startIndex int, turnIndex int) int {
	if runtime == nil {
		return 0
	}
	if startIndex < 0 || startIndex > len(runtime.MockCalls) {
		startIndex = len(runtime.MockCalls)
	}
	for _, call := range runtime.MockCalls[startIndex:] {
		data := map[string]any{"target": call.Target, "arguments": call.Arguments}
		if call.Error != "" {
			data["error"] = call.Error
		} else {
			data["result"] = call.Result
		}
		record("mock_call", turnIndex, data)
	}
	return len(runtime.MockCalls)
}

func executeEnvironment(ctx context.Context, projectRoot string, environment Environment, execute func(transcriptRecorder) (environmentExecution, error)) (SimulationResult, error) {
	runID := fmt.Sprintf("go-%d", time.Now().UnixNano())
	transcriptPath := filepath.Join(".relai", "runs", runID, "transcript.jsonl")
	absoluteTranscriptPath := filepath.Join(projectRoot, transcriptPath)
	if err := os.MkdirAll(filepath.Dir(absoluteTranscriptPath), 0o755); err != nil {
		return SimulationResult{}, err
	}

	events := []TranscriptEvent{}
	record := transcriptRecorder(func(eventType string, turnIndex int, data map[string]any) {
		events = append(events, TranscriptEvent{
			EventType: eventType,
			TurnIndex: turnIndex,
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Data:      data,
		})
	})
	record("run_start", 0, map[string]any{"target_type": targetType(environment), "input_type": inputType(environment)})

	execution, executionErr := execute(record)
	if executionErr != nil && !hasTerminalErrorEvent(events) {
		recordExecutionError(record, environment, executionErr)
	}
	record("run_end", 0, map[string]any{"reason": execution.stopReason})

	if execution.timings.ComponentRuns == nil {
		execution.timings.ComponentRuns = []ComponentRunTiming{}
	}
	result := SimulationResult{
		SchemaVersion:         SimulationResultSchemaVersion,
		LearningEnvironmentID: environment.ID,
		RunID:                 runID,
		TranscriptPath:        transcriptPath,
		EventsCount:           len(events),
		FinalOutput:           execution.finalOutput,
		StopReason:            execution.stopReason,
		Evaluations:           []EvaluationEntry{},
		EvaluationErrors:      []EvaluationError{},
		Timings:               &execution.timings,
		Metadata:              targetMetadata(environment),
		Events:                events,
	}
	for key, value := range execution.metadata {
		result.Metadata[key] = value
	}
	if err := writeJSONLines(absoluteTranscriptPath, events); err != nil {
		if executionErr != nil {
			return result, errors.Join(executionErr, fmt.Errorf("write transcript: %w", err))
		}
		return result, err
	}
	if executionErr != nil {
		return result, executionErr
	}
	for _, evaluator := range environment.Evaluators {
		evaluationTarget, isGlobalEvaluator := unwrapGlobalEvaluator(evaluator)
		minScore, maxScore, boundsErr := evaluatorScoreBounds(evaluationTarget)
		if boundsErr != nil {
			result.EvaluationErrors = append(result.EvaluationErrors, EvaluationError{
				EvaluatorID:       evaluationTarget.ID(),
				Message:           boundsErr.Error(),
				IsGlobalEvaluator: isGlobalEvaluator,
			})
			continue
		}
		var evaluation EvaluationResult
		var err error
		if contextual, ok := evaluationTarget.(interface {
			EvaluateContext(context.Context, SimulationResult) (EvaluationResult, error)
		}); ok {
			evaluation, err = contextual.EvaluateContext(ctx, result)
		} else {
			evaluation, err = evaluationTarget.Evaluate(result)
		}
		if err != nil {
			result.EvaluationErrors = append(result.EvaluationErrors, EvaluationError{
				EvaluatorID:       evaluationTarget.ID(),
				Message:           err.Error(),
				IsGlobalEvaluator: isGlobalEvaluator,
			})
			continue
		}
		if err := validateEvaluationResult(evaluationTarget.ID(), evaluation, minScore, maxScore); err != nil {
			result.EvaluationErrors = append(result.EvaluationErrors, EvaluationError{
				EvaluatorID:       evaluationTarget.ID(),
				Message:           err.Error(),
				IsGlobalEvaluator: isGlobalEvaluator,
			})
			continue
		}
		result.Evaluations = append(result.Evaluations, EvaluationEntry{
			EvaluatorID:       evaluationTarget.ID(),
			Score:             evaluation.Score,
			Feedback:          evaluation.Feedback,
			MinScore:          minScore,
			MaxScore:          maxScore,
			IsGlobalEvaluator: isGlobalEvaluator,
		})
	}
	return result, nil
}

func hasTerminalErrorEvent(events []TranscriptEvent) bool {
	for _, event := range events {
		if event.EventType == "error" || event.EventType == "component_error" {
			return true
		}
	}
	return false
}

func recordExecutionError(record transcriptRecorder, environment Environment, err error) {
	if target, ok := componentTarget(environment.Target); ok {
		record("component_error", 0, map[string]any{
			"target":     target.ImportPath,
			"message":    err.Error(),
			"error_type": "Error",
		})
		return
	}
	record("error", 0, map[string]any{"error": err.Error(), "error_type": "Error"})
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func unwrapGlobalEvaluator(evaluator Evaluator) (Evaluator, bool) {
	if global, ok := evaluator.(globalEvaluator); ok {
		return global.evaluator, true
	}
	return evaluator, false
}

func runAgentInput(ctx context.Context, input Input, adapter AgentAdapter, runtime *AdapterRuntime, factory judgeModelFactory, record transcriptRecorder) (JSONValue, string, SimulationTimings, error) {
	recordedRuntimeCalls := runtimeMockCallCount(runtime)
	switch input := input.(type) {
	case FixedInput:
		return runFixedInput(ctx, input, adapter, runtime, recordedRuntimeCalls, record)
	case *FixedInput:
		if input != nil {
			return runFixedInput(ctx, *input, adapter, runtime, recordedRuntimeCalls, record)
		}
	case PersonaInput:
		return runPersonaConversation(ctx, input, factory, adapter, runtime, recordedRuntimeCalls, record)
	case *PersonaInput:
		if input != nil {
			return runPersonaConversation(ctx, *input, factory, adapter, runtime, recordedRuntimeCalls, record)
		}
	}
	return nil, "agent error", SimulationTimings{}, errors.New("go agent runner requires fixed or persona input")
}

func runFixedInput(ctx context.Context, input FixedInput, adapter AgentAdapter, runtime *AdapterRuntime, recordedRuntimeCalls int, record transcriptRecorder) (JSONValue, string, SimulationTimings, error) {
	var finalOutput JSONValue
	var totalDurationMS int64
	for index, turn := range input.Turns {
		if err := ctx.Err(); err != nil {
			record("error", index, map[string]any{"error": err.Error(), "error_type": "Error"})
			return nil, "agent error", SimulationTimings{TotalDurationMS: totalDurationMS}, err
		}
		record("user_message", index, map[string]any{"content": turn.Content})
		startedAt := time.Now()
		result, err := adapter.RunTurn(ctx, turn.Content)
		totalDurationMS += elapsedMilliseconds(startedAt)
		if err != nil {
			recordRuntimeMockCalls(record, runtime, recordedRuntimeCalls, index)
			record("error", index, map[string]any{"error": err.Error(), "error_type": "Error"})
			return nil, "agent error", SimulationTimings{TotalDurationMS: totalDurationMS}, err
		}
		if err := ctx.Err(); err != nil {
			recordRuntimeMockCalls(record, runtime, recordedRuntimeCalls, index)
			record("error", index, map[string]any{"error": err.Error(), "error_type": "Error"})
			return nil, "agent error", SimulationTimings{TotalDurationMS: totalDurationMS}, err
		}
		finalOutput = result.AssistantMessage
		record("agent_message", index, map[string]any{"content": result.AssistantMessage, "metadata": nonNilMap(result.Metadata)})
		recordedRuntimeCalls = recordRuntimeMockCalls(record, runtime, recordedRuntimeCalls, index)
	}
	return finalOutput, "complete", SimulationTimings{
		TotalDurationMS: totalDurationMS,
		ComponentRuns:   []ComponentRunTiming{},
	}, nil
}

func resolveComponentInput(ctx context.Context, target ComponentTarget, input Input, factory judgeModelFactory) (FixedComponentInput, bool, error) {
	switch input := input.(type) {
	case FixedComponentInput:
		return input, false, nil
	case *FixedComponentInput:
		if input != nil {
			return *input, false, nil
		}
	case GeneratedComponentInput:
		return generateComponentInput(ctx, target, input, factory)
	case *GeneratedComponentInput:
		if input != nil {
			return generateComponentInput(ctx, target, *input, factory)
		}
	}
	return FixedComponentInput{}, false, errors.New("go component runner requires fixed or generated component input")
}

func generateComponentInput(ctx context.Context, target ComponentTarget, input GeneratedComponentInput, factory judgeModelFactory) (FixedComponentInput, bool, error) {
	if err := ctx.Err(); err != nil {
		return FixedComponentInput{}, false, err
	}
	prompt, err := componentInputPrompt(target, input)
	if err != nil {
		return FixedComponentInput{}, false, err
	}
	output, err := runStructuredModel(
		ctx,
		input.Model,
		"relai_component_input_generator",
		"Generates a structured call for a RELAI component.",
		"Return JSON only with args (an array) and kwargs (an object). Do not include commentary.",
		prompt,
		factory,
	)
	if err != nil {
		return FixedComponentInput{}, false, fmt.Errorf("generate component input for %q: %w", target.ImportPath, err)
	}
	var generated struct {
		Args      []JSONValue          `json:"args"`
		NamedArgs map[string]JSONValue `json:"kwargs"`
	}
	if err := json.Unmarshal([]byte(output), &generated); err != nil {
		return FixedComponentInput{}, false, fmt.Errorf("generated component input for %q is not valid JSON: %w", target.ImportPath, err)
	}
	if generated.Args == nil {
		generated.Args = []JSONValue{}
	}
	if generated.NamedArgs == nil {
		generated.NamedArgs = map[string]JSONValue{}
	}
	return FixedComponentInput{Args: generated.Args, NamedArgs: generated.NamedArgs}, true, nil
}

func componentInputPrompt(target ComponentTarget, input GeneratedComponentInput) (string, error) {
	schema, err := json.Marshal(input.InputSchema)
	if err != nil {
		return "", fmt.Errorf("serialize component input schema: %w", err)
	}
	return fmt.Sprintf("Generate a call for component %q.\n\nInstructions:\n%s\n\nInput schema:\n%s", target.ImportPath, input.Instructions, schema), nil
}

func runPersonaConversation(ctx context.Context, input PersonaInput, factory judgeModelFactory, adapter AgentAdapter, runtime *AdapterRuntime, recordedRuntimeCalls int, record transcriptRecorder) (JSONValue, string, SimulationTimings, error) {
	if input.MaxTurns <= 0 {
		return nil, "persona error", SimulationTimings{}, errors.New("persona input max turns must be greater than zero")
	}
	var finalOutput JSONValue
	var agentMessage JSONValue
	var totalDurationMS int64
	for index := 0; index < input.MaxTurns; index++ {
		if err := ctx.Err(); err != nil {
			record("error", index, map[string]any{"error": err.Error(), "error_type": "Error"})
			return nil, "persona error", SimulationTimings{TotalDurationMS: totalDurationMS}, err
		}
		var content JSONValue
		generated := true
		if index == 0 && input.SeedMessage != nil {
			content = input.SeedMessage
			generated = false
		} else {
			output, err := runStructuredModel(
				ctx,
				input.Model,
				"relai_persona",
				"Simulates a user interacting with an agent.",
				"Impersonate the specified user persona. Return JSON only with content, should_stop, and optional reason. content must be JSON-serializable.",
				personaPrompt(input, agentMessage, input.MaxTurns-index),
				factory,
			)
			if err != nil {
				return nil, "persona error", SimulationTimings{TotalDurationMS: totalDurationMS}, fmt.Errorf("generate persona input: %w", err)
			}
			var generated struct {
				Content    json.RawMessage `json:"content"`
				ShouldStop bool            `json:"should_stop"`
				Reason     string          `json:"reason"`
			}
			if err := json.Unmarshal([]byte(output), &generated); err != nil {
				return nil, "persona error", SimulationTimings{TotalDurationMS: totalDurationMS}, fmt.Errorf("generated persona input is not valid JSON: %w", err)
			}
			if generated.ShouldStop {
				record("persona_stop", index, map[string]any{"reason": generated.Reason})
				break
			}
			if len(generated.Content) == 0 {
				return nil, "persona error", SimulationTimings{TotalDurationMS: totalDurationMS}, errors.New("generated persona input is missing content")
			}
			if err := json.Unmarshal(generated.Content, &content); err != nil {
				return nil, "persona error", SimulationTimings{TotalDurationMS: totalDurationMS}, fmt.Errorf("generated persona content is invalid JSON: %w", err)
			}
		}
		if generated {
			record("persona_input_generated", index, map[string]any{"content": content})
		} else {
			record("persona_seed_message", index, map[string]any{"content": content})
		}
		record("user_message", index, map[string]any{"content": content})
		startedAt := time.Now()
		result, err := adapter.RunTurn(ctx, content)
		totalDurationMS += elapsedMilliseconds(startedAt)
		if err != nil {
			recordRuntimeMockCalls(record, runtime, recordedRuntimeCalls, index)
			record("error", index, map[string]any{"error": err.Error(), "error_type": "Error"})
			return nil, "persona error", SimulationTimings{TotalDurationMS: totalDurationMS}, err
		}
		if err := ctx.Err(); err != nil {
			recordRuntimeMockCalls(record, runtime, recordedRuntimeCalls, index)
			record("error", index, map[string]any{"error": err.Error(), "error_type": "Error"})
			return nil, "persona error", SimulationTimings{TotalDurationMS: totalDurationMS}, err
		}
		agentMessage = result.AssistantMessage
		finalOutput = result.AssistantMessage
		record("agent_message", index, map[string]any{"content": result.AssistantMessage, "metadata": nonNilMap(result.Metadata)})
		recordedRuntimeCalls = recordRuntimeMockCalls(record, runtime, recordedRuntimeCalls, index)
	}
	return finalOutput, "persona complete", SimulationTimings{
		TotalDurationMS: totalDurationMS,
		ComponentRuns:   []ComponentRunTiming{},
	}, nil
}

func componentTimings(target string, durationMS int64) SimulationTimings {
	return SimulationTimings{
		TotalDurationMS: durationMS,
		ComponentRuns:   []ComponentRunTiming{{Target: target, DurationMS: durationMS}},
	}
}

func elapsedMilliseconds(startedAt time.Time) int64 {
	durationMS := time.Since(startedAt).Milliseconds()
	if durationMS < 0 {
		return 0
	}
	return durationMS
}

func personaPrompt(input PersonaInput, agentMessage JSONValue, remainingTurns int) string {
	agentResponse, _ := json.Marshal(agentMessage)
	return fmt.Sprintf("Persona:\n%s\n\nIntent:\n%s\n\nSimulated user turns remaining, including this turn: %d\n\nLatest agent response:\n%s\n\nGenerate the next user turn.", input.Persona, input.Intent, remainingTurns, agentResponse)
}

// WriteSimulationResult writes a simulation result JSON file.
func WriteSimulationResult(path string, result SimulationResult) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// EnvironmentMetadata returns the concise metadata used for local validation.
func EnvironmentMetadata(environment Environment) map[string]any {
	serialized := environment.Serialize()
	mockTargets := make([]string, 0, len(environment.Mocks))
	for target := range environment.Mocks {
		mockTargets = append(mockTargets, target)
	}
	sort.Strings(mockTargets)
	evaluatorIDs := make([]string, 0, len(environment.Evaluators))
	for _, evaluator := range environment.Evaluators {
		evaluatorIDs = append(evaluatorIDs, evaluator.ID())
	}
	return map[string]any{
		"id":                 environment.ID,
		"description":        environment.Description,
		"tags":               environment.Tags,
		"target_type":        targetType(environment),
		"target_import_path": serializedTargetImportPath(serialized),
		"input_type":         inputType(environment),
		"mock_targets":       mockTargets,
		"evaluator_ids":      evaluatorIDs,
	}
}

// EnvironmentUploadMetadata returns the metadata payload used when uploading.
func EnvironmentUploadMetadata(environment Environment) map[string]any {
	return map[string]any{
		"id":              environment.ID,
		"serialized_json": environment.Serialize(),
		"schema_version":  LearningEnvironmentSchemaVersion,
		"name":            environment.Name,
		"description":     environment.Description,
		"tags":            environment.Tags,
	}
}

// EvaluatorMetadata returns concise metadata used for local evaluator validation.
func EvaluatorMetadata(evaluator Evaluator) map[string]any {
	payload := evaluator.evaluatorPayload()
	return map[string]any{
		"id":        payload.ID,
		"type":      payload.Type,
		"scope":     defaultScope(payload.Scope),
		"min_score": payload.MinScore,
		"max_score": payload.MaxScore,
	}
}

// EvaluatorUploadMetadata returns the metadata payload used when uploading.
func EvaluatorUploadMetadata(evaluator Evaluator) map[string]any {
	payload := evaluator.evaluatorPayload()
	return map[string]any{
		"id":           payload.ID,
		"type":         payload.Type,
		"scope":        defaultScope(payload.Scope),
		"description":  payload.Description,
		"min_score":    payload.MinScore,
		"max_score":    payload.MaxScore,
		"instructions": payload.Instructions,
		"model":        payload.Model,
	}
}

// SmokeTestEvaluator validates that a global evaluator can be loaded locally.
func SmokeTestEvaluator(evaluator Evaluator) error {
	if _, _, err := evaluatorScoreBounds(evaluator); err != nil {
		return err
	}
	if evaluator.Type() != "code" {
		return nil
	}
	_, err := evaluator.Evaluate(SimulationResult{
		SchemaVersion:         SimulationResultSchemaVersion,
		LearningEnvironmentID: "relai-evaluator-smoke",
		RunID:                 "relai-evaluator-smoke",
		TranscriptPath:        ".relai/runs/relai-evaluator-smoke.jsonl",
		EventsCount:           0,
		FinalOutput:           "Synthetic final output for evaluator smoke validation.",
		StopReason:            "smoke",
		Evaluations:           []EvaluationEntry{},
		EvaluationErrors:      []EvaluationError{},
	})
	return err
}

func defaultScoreBounds(minScore, maxScore float64) (float64, float64) {
	if minScore == 0 && maxScore == 0 {
		return 0, 1
	}
	return minScore, maxScore
}

func resolveScoreBounds(minScore, maxScore float64) (float64, float64, error) {
	minScore, maxScore = defaultScoreBounds(minScore, maxScore)
	if math.IsNaN(minScore) || math.IsInf(minScore, 0) || math.IsNaN(maxScore) || math.IsInf(maxScore, 0) {
		return 0, 0, fmt.Errorf("score bounds must be finite; got [%v, %v]", minScore, maxScore)
	}
	if minScore >= maxScore {
		return 0, 0, fmt.Errorf("minimum score must be less than maximum score; got [%v, %v]", minScore, maxScore)
	}
	return minScore, maxScore, nil
}

func evaluatorScoreBounds(evaluator Evaluator) (float64, float64, error) {
	payload := evaluator.evaluatorPayload()
	minScore, maxScore, err := resolveScoreBounds(payload.MinScore, payload.MaxScore)
	if err != nil {
		return 0, 0, fmt.Errorf("evaluator %q has invalid score bounds: %w", evaluator.ID(), err)
	}
	return minScore, maxScore, nil
}

func validateEvaluationResult(evaluatorID string, evaluation EvaluationResult, minScore, maxScore float64) error {
	if math.IsNaN(evaluation.Score) || math.IsInf(evaluation.Score, 0) {
		return fmt.Errorf("evaluator %q returned non-finite score %v", evaluatorID, evaluation.Score)
	}
	if evaluation.Score < minScore || evaluation.Score > maxScore {
		return fmt.Errorf("evaluator %q returned score %v outside bounds [%v, %v]", evaluatorID, evaluation.Score, minScore, maxScore)
	}
	if strings.TrimSpace(evaluation.Feedback) == "" {
		return fmt.Errorf("evaluator %q returned empty feedback", evaluatorID)
	}
	return nil
}

// BenchmarkMetadata returns concise metadata used for local benchmark validation.
func BenchmarkMetadata(benchmark Benchmark, benchmarkPath, projectRoot string, allowSampleCountFailure bool) (map[string]any, error) {
	var sampleCount any
	samples, err := ListBenchmarkSamples(benchmark, benchmarkPath, projectRoot)
	if err != nil {
		if !allowSampleCountFailure {
			return nil, err
		}
		sampleCount = nil
	} else {
		sampleCount = len(samples)
	}
	refType, refValue := benchmarkDatasetRefSummary(benchmark)
	return map[string]any{
		"id":                benchmark.ID,
		"name":              benchmark.Name,
		"description":       benchmark.Description,
		"sample_count":      sampleCount,
		"dataset_ref_type":  refType,
		"dataset_ref_value": refValue,
	}, nil
}

// BenchmarkUploadMetadata returns the metadata payload used when uploading.
func BenchmarkUploadMetadata(benchmark Benchmark) map[string]any {
	return map[string]any{
		"benchmark_id":     benchmark.ID,
		"name":             benchmark.Name,
		"description":      benchmark.Description,
		"required_columns": benchmark.RequiredColumns,
		"schema_version":   BenchmarkSchemaVersion,
		"serialized_json":  benchmark.Serialize(),
	}
}

// BenchmarkSamplesMetadata returns sample metadata used for selection.
func BenchmarkSamplesMetadata(benchmark Benchmark, benchmarkPath, projectRoot string) (map[string]any, error) {
	samples, err := ListBenchmarkSamples(benchmark, benchmarkPath, projectRoot)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(samples))
	for _, sample := range samples {
		environment, err := ExpandBenchmarkSample(benchmark, sample)
		if err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"sample_index":   sample.Index,
			"sample_id":      sample.ID,
			"environment_id": environment.ID,
			"description":    environment.Description,
			"tags":           environment.Tags,
		})
		if target, ok := agentTarget(environment.Target); ok && target.AgentTarget != "" {
			items[len(items)-1]["agent_target"] = target.AgentTarget
		}
	}
	payload := map[string]any{"benchmark_id": benchmark.ID, "samples": items}
	if benchmark.AgentTarget != "" {
		payload["agent_target"] = benchmark.AgentTarget
	}
	return payload, nil
}

// ListBenchmarkSamples reads benchmark CSV rows and returns samples.
func ListBenchmarkSamples(benchmark Benchmark, benchmarkPath, projectRoot string) ([]BenchmarkSample, error) {
	rows, err := loadBenchmarkRows(benchmark, benchmarkPath, projectRoot)
	if err != nil {
		return nil, err
	}
	samples := make([]BenchmarkSample, 0, len(rows))
	seen := map[string]struct{}{}
	for index, row := range rows {
		sampleID := strconv.Itoa(index)
		if benchmark.SampleIDColumn != "" {
			value := row[benchmark.SampleIDColumn]
			if value == "" {
				return nil, fmt.Errorf("benchmark sample_id_column %q is empty at row %d", benchmark.SampleIDColumn, index)
			}
			sampleID = value
		}
		if _, exists := seen[sampleID]; exists {
			return nil, fmt.Errorf("benchmark sample_id %q is duplicated", sampleID)
		}
		seen[sampleID] = struct{}{}
		samples = append(samples, BenchmarkSample{Index: index, ID: sampleID, Row: row})
	}
	return samples, nil
}

// GetBenchmarkSample returns one benchmark CSV row by sample ID.
func GetBenchmarkSample(benchmark Benchmark, sampleID, benchmarkPath, projectRoot string) (BenchmarkSample, error) {
	samples, err := ListBenchmarkSamples(benchmark, benchmarkPath, projectRoot)
	if err != nil {
		return BenchmarkSample{}, err
	}
	for _, sample := range samples {
		if sample.ID == sampleID {
			return sample, nil
		}
	}
	return BenchmarkSample{}, fmt.Errorf("benchmark sample %q not found", sampleID)
}

// ExpandBenchmarkSample builds and annotates one benchmark sample environment.
func ExpandBenchmarkSample(benchmark Benchmark, sample BenchmarkSample) (Environment, error) {
	if benchmark.BuildEnvironment == nil {
		return Environment{}, errors.New("benchmark requires BuildEnvironment")
	}
	environment, err := benchmark.BuildEnvironment(sample)
	if err != nil {
		return Environment{}, err
	}
	if target, ok := agentTarget(environment.Target); ok && target.AgentTarget == "" && benchmark.AgentTarget != "" {
		environment.Target = AgentTarget{AgentTarget: benchmark.AgentTarget}
	} else if target, ok := agentTarget(environment.Target); ok && benchmark.AgentTarget != "" && target.AgentTarget != benchmark.AgentTarget {
		return Environment{}, fmt.Errorf("benchmark agent target %q does not match environment agent target %q", benchmark.AgentTarget, target.AgentTarget)
	}
	tag := fmt.Sprintf("benchmark-%s-sample-%d", benchmark.ID, sample.Index)
	if !containsString(environment.Tags, tag) {
		environment.Tags = append(environment.Tags, tag)
	}
	return environment, nil
}

func serializeEvaluators(evaluators []Evaluator) []EvaluatorPayload {
	values := make([]EvaluatorPayload, 0, len(evaluators))
	for _, evaluator := range evaluators {
		values = append(values, evaluator.evaluatorPayload())
	}
	return values
}

func defaultScope(scope string) string {
	if scope == "" {
		return "end-to-end"
	}
	return scope
}

func benchmarkDatasetRefSummary(benchmark Benchmark) (string, string) {
	refType := stringFromDatasetRef(benchmark.DatasetRef, "type")
	switch refType {
	case "stored_csv":
		return refType, stringFromDatasetRef(benchmark.DatasetRef, "id")
	case "local_csv":
		return refType, stringFromDatasetRef(benchmark.DatasetRef, "path")
	default:
		return refType, ""
	}
}

func loadBenchmarkRows(benchmark Benchmark, benchmarkPath, projectRoot string) ([]map[string]string, error) {
	datasetPath, err := resolveBenchmarkDatasetPath(benchmark, benchmarkPath, projectRoot)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(datasetPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, errors.New("benchmark CSV must include headers and at least one row")
	}
	headers := records[0]
	headerIndex := make(map[string]int, len(headers))
	for index, header := range headers {
		headerIndex[header] = index
	}
	for _, column := range benchmark.RequiredColumns {
		if _, ok := headerIndex[column]; !ok {
			return nil, fmt.Errorf("benchmark CSV is missing required column %s", column)
		}
	}
	rows := make([]map[string]string, 0, len(records)-1)
	for _, record := range records[1:] {
		if emptyCSVRecord(record) {
			continue
		}
		row := make(map[string]string, len(headers))
		for index, header := range headers {
			if index < len(record) {
				row[header] = record[index]
			} else {
				row[header] = ""
			}
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, io.ErrUnexpectedEOF
	}
	return rows, nil
}

func resolveBenchmarkDatasetPath(benchmark Benchmark, benchmarkPath, projectRoot string) (string, error) {
	refType, refValue := benchmarkDatasetRefSummary(benchmark)
	switch refType {
	case "local_csv":
		if filepath.IsAbs(refValue) {
			return refValue, nil
		}
		base := "."
		if benchmarkPath != "" {
			base = filepath.Dir(benchmarkPath)
		}
		return filepath.Join(base, refValue), nil
	case "stored_csv":
		if projectRoot == "" {
			return "", errors.New("stored benchmark CSV requires project root")
		}
		return filepath.Join(projectRoot, ".relai", "benchmark_csv", refValue+".csv"), nil
	default:
		return "", fmt.Errorf("unsupported benchmark dataset_ref type %q", refType)
	}
}

func stringFromDatasetRef(ref map[string]any, key string) string {
	if ref == nil {
		return ""
	}
	if _, ok := ref[key]; !ok {
		return ""
	}
	if value, ok := ref[key].(string); ok {
		return value
	}
	return fmt.Sprint(ref[key])
}

func emptyCSVRecord(record []string) bool {
	for _, value := range record {
		if value != "" {
			return false
		}
	}
	return true
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func targetType(environment Environment) string {
	if environment.Target == nil {
		return "agent"
	}
	return environment.Target.Type()
}

func componentTarget(target Target) (ComponentTarget, bool) {
	switch target := target.(type) {
	case ComponentTarget:
		return target, true
	case *ComponentTarget:
		if target != nil {
			return *target, true
		}
	}
	return ComponentTarget{}, false
}

func agentTarget(target Target) (AgentTarget, bool) {
	switch target := target.(type) {
	case AgentTarget:
		return target, true
	case *AgentTarget:
		if target != nil {
			return *target, true
		}
	}
	return AgentTarget{}, false
}

func targetMetadata(environment Environment) map[string]any {
	metadata := map[string]any{"target": targetType(environment)}
	if target, ok := agentTarget(environment.Target); ok && target.AgentTarget != "" {
		metadata["agent_target"] = target.AgentTarget
	}
	return metadata
}

func inputType(environment Environment) string {
	if environment.Input == nil {
		return "none"
	}
	return environment.Input.Type()
}

func serializedTargetImportPath(serialized EnvironmentPayload) any {
	if serialized.Target.Type != "component" {
		return nil
	}
	return serialized.Target.ImportPath
}

func nonNilMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func writeJSONLines(path string, events []TranscriptEvent) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return err
		}
	}
	return nil
}
