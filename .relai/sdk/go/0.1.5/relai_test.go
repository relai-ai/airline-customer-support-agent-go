package relai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/genai"

	adkmodel "google.golang.org/adk/model"
)

type echoAdapter struct{}

func (echoAdapter) AdapterCapabilities() []AdapterCapability {
	return []AdapterCapability{AdapterCapabilityRunTurn}
}

type componentEchoAdapter struct{}

type failingAgentAdapter struct {
	err error
}

type failingComponentAdapter struct {
	err error
}

type failingTargetConfigurer struct {
	err error
}

type targetSelectingAdapter struct {
	echoAdapter
	target string
}

type runtimeRecordingAgentAdapter struct {
	runtime *AdapterRuntime
	turn    int
	failAt  int
}

type runtimeRecordingComponentAdapter struct {
	runtime *AdapterRuntime
	err     error
}

type staticJudgeModel struct {
	response            string
	expectedInstruction string
}

func (m staticJudgeModel) Name() string { return "static-judge" }

func (m staticJudgeModel) GenerateContent(_ context.Context, req *adkmodel.LLMRequest, _ bool) iter.Seq2[*adkmodel.LLMResponse, error] {
	return func(yield func(*adkmodel.LLMResponse, error) bool) {
		if req.Config == nil || req.Config.ResponseMIMEType != "application/json" {
			yield(nil, errors.New("judge did not request JSON output"))
			return
		}
		if m.expectedInstruction != "" && (req.Config.SystemInstruction == nil || !strings.Contains(contentText(req.Config.SystemInstruction), m.expectedInstruction)) {
			yield(nil, errors.New("judge instruction was not passed through ADK"))
			return
		}
		yield(&adkmodel.LLMResponse{Content: genai.NewContentFromText(m.response, genai.RoleModel)}, nil)
	}
}

func (echoAdapter) RunTurn(_ context.Context, input JSONValue) (AgentTurnResult, error) {
	return AgentTurnResult{AssistantMessage: input}, nil
}

func TestAdapterCapabilitiesRequireRunTurnAndVerifyOptionalBehavior(t *testing.T) {
	adapter := echoAdapter{}
	capabilities, err := ValidateAdapterCapabilities(adapter)
	if err != nil {
		t.Fatal(err)
	}
	capabilities[AdapterCapabilityToolOverrides] = struct{}{}
	if err := ValidateAdapterSelfCheck(capabilities, map[AdapterCapability]bool{
		AdapterCapabilityToolOverrides: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAdapterSelfCheck(capabilities, map[AdapterCapability]bool{}); err == nil {
		t.Fatal("expected an unverified optional capability to fail")
	}
}

func (adapter *targetSelectingAdapter) ConfigureAgentTarget(_ context.Context, target AgentTarget) error {
	adapter.target = target.AgentTarget
	return nil
}

func (componentEchoAdapter) RunComponent(_ context.Context, target ComponentTarget, input FixedComponentInput) (JSONValue, error) {
	return map[string]JSONValue{"target": target.ImportPath, "arg": input.Args[0], "city": input.NamedArgs["city"]}, nil
}

func (adapter failingAgentAdapter) RunTurn(context.Context, JSONValue) (AgentTurnResult, error) {
	return AgentTurnResult{}, adapter.err
}

func (adapter failingComponentAdapter) RunComponent(context.Context, ComponentTarget, FixedComponentInput) (JSONValue, error) {
	return nil, adapter.err
}

func (adapter *runtimeRecordingAgentAdapter) RunTurn(_ context.Context, input JSONValue) (AgentTurnResult, error) {
	call := MockCallRecord{Target: fmt.Sprintf("tool-%d", adapter.turn), Arguments: input, Result: "ok"}
	if adapter.turn == adapter.failAt {
		call.Result = nil
		call.Error = "override failed"
	}
	adapter.runtime.RecordMockCall(call)
	adapter.turn++
	if call.Error != "" {
		return AgentTurnResult{}, errors.New(call.Error)
	}
	return AgentTurnResult{AssistantMessage: input}, nil
}

func (adapter runtimeRecordingComponentAdapter) RunComponent(_ context.Context, _ ComponentTarget, input FixedComponentInput) (JSONValue, error) {
	call := MockCallRecord{Target: "component-tool", Arguments: input.Serialize(), Result: "ok"}
	if adapter.err != nil {
		call.Result = nil
		call.Error = adapter.err.Error()
	}
	adapter.runtime.RecordMockCall(call)
	return call.Result, adapter.err
}

func (adapter failingTargetConfigurer) ConfigureAgentTarget(context.Context, AgentTarget) error {
	return adapter.err
}

func (failingTargetConfigurer) RunTurn(context.Context, JSONValue) (AgentTurnResult, error) {
	return AgentTurnResult{}, errors.New("agent should not run after setup failure")
}

func assertPersistedEventTypes(t *testing.T, projectRoot string, result SimulationResult, want ...string) {
	t.Helper()
	if result.RunID == "" || result.TranscriptPath == "" {
		t.Fatalf("expected populated partial result, got %#v", result)
	}
	if result.EventsCount != len(want) || len(result.Events) != len(want) {
		t.Fatalf("unexpected in-memory events: count=%d events=%#v", result.EventsCount, result.Events)
	}
	for index, eventType := range want {
		if result.Events[index].EventType != eventType {
			t.Fatalf("event %d: got %q, want %q", index, result.Events[index].EventType, eventType)
		}
	}

	contents, err := os.ReadFile(filepath.Join(projectRoot, result.TranscriptPath))
	if err != nil {
		t.Fatalf("read persisted transcript: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(contents)), "\n")
	if len(lines) != len(want) {
		t.Fatalf("persisted transcript has %d events, want %d: %s", len(lines), len(want), contents)
	}
	for index, line := range lines {
		var event TranscriptEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("decode persisted event %d: %v", index, err)
		}
		if event.EventType != want[index] {
			t.Fatalf("persisted event %d: got %q, want %q", index, event.EventType, want[index])
		}
	}
}

func mustNotRunEvaluator(invoked *bool) Evaluator {
	return CodeEvaluator{
		EvaluatorID: "must-not-run",
		Fn: func(SimulationResult) (EvaluationResult, error) {
			*invoked = true
			return EvaluationResult{Score: 1, Feedback: "unexpected"}, nil
		},
	}
}

func TestFixedComponentInputUsesNamedArgsForWireNamedArgs(t *testing.T) {
	payload := FixedComponentInput{
		Args: []JSONValue{"weather"},
		NamedArgs: map[string]JSONValue{
			"city": "Paris",
		},
	}.Serialize()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(data)
	if !strings.Contains(jsonText, `"kwargs":{"city":"Paris"}`) {
		t.Fatalf("expected named args to serialize as kwargs, got %s", jsonText)
	}
	if strings.Contains(jsonText, "NamedArgs") {
		t.Fatalf("payload leaked Go field name: %s", jsonText)
	}
}

func TestCodeEvaluatorPayloadOmitsModel(t *testing.T) {
	payload := CodeEvaluator{
		EvaluatorID: "contains-hello",
		Description: "Checks output.",
		MinScore:    0,
		MaxScore:    1,
	}.Serialize()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "model") {
		t.Fatalf("code evaluator payload should not include model: %s", string(data))
	}
}

func TestCodeEvaluatorPayloadResolvesScoreBounds(t *testing.T) {
	for _, test := range []struct {
		name         string
		minScore     float64
		maxScore     float64
		wantMinScore float64
		wantMaxScore float64
	}{
		{name: "omitted", wantMinScore: 0, wantMaxScore: 1},
		{name: "custom", minScore: -2, maxScore: 3, wantMinScore: -2, wantMaxScore: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			payload := CodeEvaluator{MinScore: test.minScore, MaxScore: test.maxScore}.Serialize()
			if payload.MinScore != test.wantMinScore || payload.MaxScore != test.wantMaxScore {
				t.Fatalf("unexpected score bounds: %v to %v", payload.MinScore, payload.MaxScore)
			}
		})
	}
}

func TestEvaluatorScoreBoundsRejectMalformedConfiguration(t *testing.T) {
	for _, test := range []struct {
		name     string
		minScore float64
		maxScore float64
	}{
		{name: "nan minimum", minScore: math.NaN(), maxScore: 1},
		{name: "infinite minimum", minScore: math.Inf(-1), maxScore: 1},
		{name: "infinite maximum", minScore: 0, maxScore: math.Inf(1)},
		{name: "equal", minScore: 1, maxScore: 1},
		{name: "reversed", minScore: 2, maxScore: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			invoked := false
			evaluator := CodeEvaluator{
				EvaluatorID: "invalid-bounds",
				MinScore:    test.minScore,
				MaxScore:    test.maxScore,
				Fn: func(SimulationResult) (EvaluationResult, error) {
					invoked = true
					return EvaluationResult{Score: 1, Feedback: "ok"}, nil
				},
			}
			if _, err := evaluator.Evaluate(SimulationResult{}); err == nil {
				t.Fatal("expected malformed bounds to fail")
			}
			if invoked {
				t.Fatal("evaluator function ran despite malformed bounds")
			}
		})
	}
}

func TestCodeEvaluatorRejectsInvalidResultsAndPreservesValidFeedback(t *testing.T) {
	for _, test := range []struct {
		name       string
		evaluation EvaluationResult
		wantError  bool
	}{
		{name: "nan", evaluation: EvaluationResult{Score: math.NaN(), Feedback: "bad"}, wantError: true},
		{name: "positive infinity", evaluation: EvaluationResult{Score: math.Inf(1), Feedback: "bad"}, wantError: true},
		{name: "negative infinity", evaluation: EvaluationResult{Score: math.Inf(-1), Feedback: "bad"}, wantError: true},
		{name: "below bounds", evaluation: EvaluationResult{Score: -1, Feedback: "bad"}, wantError: true},
		{name: "above bounds", evaluation: EvaluationResult{Score: 2, Feedback: "bad"}, wantError: true},
		{name: "empty feedback", evaluation: EvaluationResult{Score: 0.5}, wantError: true},
		{name: "blank feedback", evaluation: EvaluationResult{Score: 0.5, Feedback: " \t\n"}, wantError: true},
		{name: "minimum inclusive", evaluation: EvaluationResult{Score: 0, Feedback: "minimum"}},
		{name: "maximum inclusive", evaluation: EvaluationResult{Score: 1, Feedback: "  keep spacing  "}},
	} {
		t.Run(test.name, func(t *testing.T) {
			evaluator := CodeEvaluator{
				EvaluatorID: "result-check",
				Fn:          func(SimulationResult) (EvaluationResult, error) { return test.evaluation, nil },
			}
			got, err := evaluator.Evaluate(SimulationResult{})
			if (err != nil) != test.wantError {
				t.Fatalf("unexpected error: %v", err)
			}
			if !test.wantError && got.Feedback != test.evaluation.Feedback {
				t.Fatalf("feedback changed from %q to %q", test.evaluation.Feedback, got.Feedback)
			}
		})
	}
}

func TestLLMJudgePayloadDefaultsScoreBounds(t *testing.T) {
	payload := LLMJudgeEvaluator{
		EvaluatorID:  "quality",
		Description:  "Checks quality.",
		Instructions: "Return a score.",
		Model:        ModelSpec{Name: "judge-model"},
	}.Serialize()
	if payload.MinScore != 0 || payload.MaxScore != 1 {
		t.Fatalf("unexpected default score bounds: %v to %v", payload.MinScore, payload.MaxScore)
	}
}

func TestEnvironmentSerializesStableShape(t *testing.T) {
	environment := Environment{
		ID:          "support-smoke",
		Name:        "Support smoke",
		Description: "Checks one support turn.",
		Tags:        []string{"end-to-end"},
		Target:      AgentTarget{},
		Input:       FixedInput{Turns: []FixedTurn{{Content: "hello"}}},
		Evaluators: []Evaluator{
			CodeEvaluator{
				EvaluatorID: "contains-hello",
				Description: "Checks output.",
				MinScore:    0,
				MaxScore:    1,
				Fn: func(SimulationResult) (EvaluationResult, error) {
					return EvaluationResult{Score: 1, Feedback: "ok"}, nil
				},
			},
		},
	}

	serialized := environment.Serialize()
	if serialized.SchemaVersion != LearningEnvironmentSchemaVersion {
		t.Fatalf("unexpected schema version: %v", serialized.SchemaVersion)
	}
	if serialized.ID != "support-smoke" {
		t.Fatalf("unexpected id: %v", serialized.ID)
	}
}

func TestAgentTargetSerializesAndConfiguresAdapter(t *testing.T) {
	target := AgentTarget{AgentTarget: "billing"}
	if payload := target.Serialize(); payload.AgentTarget != "billing" {
		t.Fatalf("expected agent target payload, got %#v", payload)
	}

	projectRoot := t.TempDir()
	adapter := &targetSelectingAdapter{}
	_, err := RunEnvironment(context.Background(), projectRoot, Environment{
		ID:          "targeted-agent",
		Name:        "Targeted agent",
		Description: "Selects a named agent.",
		Target:      target,
		Input:       FixedInput{Turns: []FixedTurn{{Content: "hello"}}},
	}, adapter)
	if err != nil {
		t.Fatal(err)
	}
	if adapter.target != "billing" {
		t.Fatalf("expected selected target billing, got %q", adapter.target)
	}
}

func TestBenchmarkAgentTargetAppliesToDefaultAgentSample(t *testing.T) {
	benchmark := Benchmark{
		ID:          "targeted-suite",
		Name:        "Targeted suite",
		Description: "Selects a shared runtime agent.",
		AgentTarget: "billing",
		BuildEnvironment: func(sample BenchmarkSample) (Environment, error) {
			return Environment{
				ID:          sample.ID,
				Name:        "Targeted sample",
				Description: "Uses the benchmark default target.",
				Target:      AgentTarget{},
				Input:       FixedInput{Turns: []FixedTurn{{Content: "hello"}}},
			}, nil
		},
	}
	environment, err := ExpandBenchmarkSample(benchmark, BenchmarkSample{Index: 0, ID: "row-1"})
	if err != nil {
		t.Fatal(err)
	}
	target, ok := agentTarget(environment.Target)
	if !ok || target.AgentTarget != "billing" {
		t.Fatalf("expected benchmark target, got %#v", environment.Target)
	}
}

func TestRunEnvironmentProducesSimulationResult(t *testing.T) {
	environment := Environment{
		ID:          "support-smoke",
		Name:        "Support smoke",
		Description: "Checks one support turn.",
		Target:      AgentTarget{},
		Input:       FixedInput{Turns: []FixedTurn{{Content: "hello"}}},
	}

	result, err := RunEnvironment(context.Background(), t.TempDir(), environment, echoAdapter{})
	if err != nil {
		t.Fatal(err)
	}
	if result.LearningEnvironmentID != "support-smoke" {
		t.Fatalf("unexpected learning environment id: %s", result.LearningEnvironmentID)
	}
	if result.EventsCount != 4 {
		t.Fatalf("unexpected events count: %d", result.EventsCount)
	}
	if result.FinalOutput != "hello" {
		t.Fatalf("unexpected final output: %v", result.FinalOutput)
	}
	if result.Timings == nil || result.Timings.TotalDurationMS < 0 || len(result.Timings.ComponentRuns) != 0 {
		t.Fatalf("unexpected agent timings: %#v", result.Timings)
	}
	serialized, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(serialized), `"timings":{"total_duration_ms":`) {
		t.Fatalf("expected timing payload, got %s", serialized)
	}
	if !strings.Contains(string(serialized), `"component_runs":[]`) {
		t.Fatalf("expected empty component runs array, got %s", serialized)
	}
}

func TestRuntimeMockCallsPreserveExecutionBoundaries(t *testing.T) {
	t.Run("agent turns exclude pre-run calls and retain turn indexes", func(t *testing.T) {
		runtime := NewAdapterRuntime(nil)
		runtime.RecordMockCall(MockCallRecord{Target: "self-check", Result: "ok"})
		adapter := &runtimeRecordingAgentAdapter{runtime: runtime, failAt: -1}
		environment := Environment{
			ID:          "runtime-turns",
			Name:        "Runtime turns",
			Description: "Records named override calls per turn.",
			Target:      AgentTarget{},
			Input: FixedInput{Turns: []FixedTurn{
				{Content: "first"},
				{Content: "second"},
			}},
		}

		result, err := RunEnvironmentWithRuntime(context.Background(), t.TempDir(), environment, adapter, runtime)
		if err != nil {
			t.Fatal(err)
		}
		mockEvents := []TranscriptEvent{}
		for _, event := range result.Events {
			if event.EventType == "mock_call" {
				mockEvents = append(mockEvents, event)
			}
		}
		if len(mockEvents) != 2 {
			t.Fatalf("expected only per-turn runtime calls, got %#v", mockEvents)
		}
		for index, event := range mockEvents {
			if event.TurnIndex != index || event.Data["target"] != fmt.Sprintf("tool-%d", index) {
				t.Fatalf("runtime call %d was attributed incorrectly: %#v", index, event)
			}
		}
	})

	t.Run("failed agent override is recorded before the terminal error", func(t *testing.T) {
		runtime := NewAdapterRuntime(nil)
		adapter := &runtimeRecordingAgentAdapter{runtime: runtime, failAt: 0}
		environment := Environment{
			ID:          "runtime-error",
			Name:        "Runtime error",
			Description: "Records a failed named override call.",
			Target:      AgentTarget{},
			Input:       FixedInput{Turns: []FixedTurn{{Content: "fail"}}},
		}

		projectRoot := t.TempDir()
		result, err := RunEnvironmentWithRuntime(context.Background(), projectRoot, environment, adapter, runtime)
		if err == nil {
			t.Fatal("expected adapter failure")
		}
		assertPersistedEventTypes(t, projectRoot, result, "run_start", "user_message", "mock_call", "error", "run_end")
		if len(result.Events) != 5 || result.Events[2].EventType != "mock_call" || result.Events[3].EventType != "error" {
			t.Fatalf("expected failed mock call before terminal error, got %#v", result.Events)
		}
		if result.Events[2].Data["error"] != "override failed" {
			t.Fatalf("expected override failure evidence, got %#v", result.Events[2])
		}
	})

	t.Run("component override calls are recorded on failure", func(t *testing.T) {
		runtime := NewAdapterRuntime(nil)
		adapter := runtimeRecordingComponentAdapter{runtime: runtime, err: errors.New("component override failed")}
		environment := Environment{
			ID:          "component-runtime-error",
			Name:        "Component runtime error",
			Description: "Records a failed component override call.",
			Target:      ComponentTarget{ImportPath: "tools:component"},
			Input:       FixedComponentInput{Args: []JSONValue{"input"}},
		}

		result, err := RunComponentEnvironmentWithRuntime(context.Background(), t.TempDir(), environment, adapter, runtime)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Events) != 5 || result.Events[2].EventType != "mock_call" || result.Events[3].EventType != "component_error" {
			t.Fatalf("expected component mock call before component error, got %#v", result.Events)
		}
		if result.Events[2].Data["error"] != "component override failed" {
			t.Fatalf("expected component override failure evidence, got %#v", result.Events[2])
		}
	})
}

func TestRunEnvironmentPersistsTranscriptForAgentFailure(t *testing.T) {
	projectRoot := t.TempDir()
	evaluatorInvoked := false
	environment := Environment{
		ID:          "agent-failure",
		Name:        "Agent failure",
		Description: "Captures an agent adapter failure.",
		Target:      AgentTarget{},
		Input:       FixedInput{Turns: []FixedTurn{{Content: "hello"}}},
		Evaluators:  []Evaluator{mustNotRunEvaluator(&evaluatorInvoked)},
	}

	result, err := RunEnvironment(context.Background(), projectRoot, environment, failingAgentAdapter{err: errors.New("agent unavailable")})
	if err == nil || !strings.Contains(err.Error(), "agent unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
	if evaluatorInvoked {
		t.Fatal("evaluator ran after an operational agent failure")
	}
	if result.StopReason != "agent error" || result.Timings == nil {
		t.Fatalf("unexpected partial result: %#v", result)
	}
	assertPersistedEventTypes(t, projectRoot, result, "run_start", "user_message", "error", "run_end")
	if got := result.Events[2].Data["error"]; got != "agent unavailable" {
		t.Fatalf("unexpected terminal error event: %#v", result.Events[2])
	}
}

func TestRunComponentEnvironmentTreatsInvocationFailureAsResult(t *testing.T) {
	projectRoot := t.TempDir()
	evaluatorInvoked := false
	environment := Environment{
		ID:          "component-failure",
		Name:        "Component failure",
		Description: "Captures a component invocation failure.",
		Target:      ComponentTarget{ImportPath: "tools:explode"},
		Input:       FixedComponentInput{Args: []JSONValue{}},
		Evaluators: []Evaluator{CodeEvaluator{
			EvaluatorID: "saw-component-error",
			Fn: func(result SimulationResult) (EvaluationResult, error) {
				evaluatorInvoked = true
				for _, event := range result.Events {
					if event.EventType == "component_error" {
						return EvaluationResult{Score: 1, Feedback: "captured"}, nil
					}
				}
				return EvaluationResult{Score: 0, Feedback: "missing"}, nil
			},
		}},
	}

	result, err := RunComponentEnvironment(context.Background(), projectRoot, environment, failingComponentAdapter{err: errors.New("component exploded")})
	if err != nil {
		t.Fatalf("component invocation failure returned an operational error: %v", err)
	}
	if !evaluatorInvoked || len(result.Evaluations) != 1 || result.Evaluations[0].Feedback != "captured" {
		t.Fatalf("component failure evaluator did not run: %#v", result.Evaluations)
	}
	if result.StopReason != "component error" || result.FinalOutput != nil {
		t.Fatalf("unexpected component failure result: %#v", result)
	}
	componentError, ok := result.Metadata["component_error"].(map[string]any)
	if !ok || componentError["message"] != "component exploded" || componentError["error_type"] != "Error" {
		t.Fatalf("unexpected component failure metadata: %#v", result.Metadata)
	}
	assertPersistedEventTypes(t, projectRoot, result, "run_start", "component_call", "component_error", "run_end")

	resultPath := filepath.Join(projectRoot, "result.json")
	if err := WriteSimulationResult(resultPath, result); err != nil {
		t.Fatalf("write component failure result: %v", err)
	}
	if _, err := os.Stat(resultPath); err != nil {
		t.Fatalf("component failure result JSON was not written: %v", err)
	}
}

func TestRunEnvironmentPersistsTranscriptForGenerationFailures(t *testing.T) {
	failingFactory := func(context.Context, ModelSpec) (adkmodel.LLM, error) {
		return nil, errors.New("generator unavailable")
	}

	t.Run("persona", func(t *testing.T) {
		projectRoot := t.TempDir()
		evaluatorInvoked := false
		result, err := runAgentEnvironmentWithModelFactory(context.Background(), projectRoot, Environment{
			ID:          "persona-generation-failure",
			Name:        "Persona generation failure",
			Description: "Captures failed persona generation.",
			Target:      AgentTarget{},
			Input:       PersonaInput{Persona: "A user", MaxTurns: 1, Model: ModelSpec{Name: "test-model"}},
			Evaluators:  []Evaluator{mustNotRunEvaluator(&evaluatorInvoked)},
		}, echoAdapter{}, nil, failingFactory)
		if err == nil || !strings.Contains(err.Error(), "generate persona input") {
			t.Fatalf("unexpected error: %v", err)
		}
		if evaluatorInvoked {
			t.Fatal("evaluator ran after operational persona generation failure")
		}
		assertPersistedEventTypes(t, projectRoot, result, "run_start", "error", "run_end")
	})

	t.Run("component", func(t *testing.T) {
		projectRoot := t.TempDir()
		evaluatorInvoked := false
		result, err := runComponentEnvironmentWithModelFactory(context.Background(), projectRoot, Environment{
			ID:          "component-generation-failure",
			Name:        "Component generation failure",
			Description: "Captures failed component input generation.",
			Target:      ComponentTarget{ImportPath: "tools:lookup"},
			Input: GeneratedComponentInput{
				Instructions: "Generate an input.",
				Model:        ModelSpec{Name: "test-model"},
				InputSchema:  map[string]JSONValue{"type": "object"},
			},
			Evaluators: []Evaluator{mustNotRunEvaluator(&evaluatorInvoked)},
		}, componentEchoAdapter{}, nil, failingFactory)
		if err == nil || !strings.Contains(err.Error(), "generate component input") {
			t.Fatalf("unexpected error: %v", err)
		}
		if evaluatorInvoked {
			t.Fatal("evaluator ran after operational component generation failure")
		}
		assertPersistedEventTypes(t, projectRoot, result, "run_start", "component_error", "run_end")
	})
}

func TestRunComponentEnvironmentPersistsTranscriptForCancellation(t *testing.T) {
	projectRoot := t.TempDir()
	evaluatorInvoked := false
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	environment := Environment{
		ID:          "component-cancelled",
		Name:        "Component cancelled",
		Description: "Captures context cancellation.",
		Target:      ComponentTarget{ImportPath: "tools:lookup"},
		Input:       FixedComponentInput{Args: []JSONValue{"weather"}},
		Evaluators:  []Evaluator{mustNotRunEvaluator(&evaluatorInvoked)},
	}

	result, err := RunComponentEnvironment(ctx, projectRoot, environment, componentEchoAdapter{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if evaluatorInvoked {
		t.Fatal("evaluator ran after operational context cancellation")
	}
	assertPersistedEventTypes(t, projectRoot, result, "run_start", "component_error", "run_end")
	if got := result.Events[1].Data["message"]; got != context.Canceled.Error() {
		t.Fatalf("unexpected cancellation event: %#v", result.Events[1])
	}
}

func TestRunEnvironmentPersistsTranscriptForAdapterSetupFailure(t *testing.T) {
	projectRoot := t.TempDir()
	evaluatorInvoked := false
	environment := Environment{
		ID:          "agent-setup-failure",
		Name:        "Agent setup failure",
		Description: "Captures adapter setup failure.",
		Target:      AgentTarget{AgentTarget: "billing"},
		Input:       FixedInput{Turns: []FixedTurn{{Content: "hello"}}},
		Evaluators:  []Evaluator{mustNotRunEvaluator(&evaluatorInvoked)},
	}

	result, err := RunEnvironment(context.Background(), projectRoot, environment, failingTargetConfigurer{err: errors.New("configuration failed")})
	if err == nil || !strings.Contains(err.Error(), "configure agent target") {
		t.Fatalf("unexpected error: %v", err)
	}
	if evaluatorInvoked {
		t.Fatal("evaluator ran after operational adapter setup failure")
	}
	assertPersistedEventTypes(t, projectRoot, result, "run_start", "error", "run_end")
}

func TestRunEnvironmentRecordsInvalidEvaluatorOutputAndContinues(t *testing.T) {
	invalidInvoked := false
	environment := Environment{
		ID:          "evaluator-validation",
		Name:        "Evaluator validation",
		Description: "Checks evaluator validation.",
		Target:      AgentTarget{},
		Input:       FixedInput{Turns: []FixedTurn{{Content: "hello"}}},
		Evaluators: []Evaluator{
			CodeEvaluator{
				EvaluatorID: "invalid-bounds",
				MinScore:    2,
				MaxScore:    1,
				Fn: func(SimulationResult) (EvaluationResult, error) {
					invalidInvoked = true
					return EvaluationResult{Score: 1, Feedback: "should not run"}, nil
				},
			},
			CodeEvaluator{
				EvaluatorID: "invalid-result",
				Fn: func(SimulationResult) (EvaluationResult, error) {
					return EvaluationResult{Score: math.NaN(), Feedback: "invalid"}, nil
				},
			},
			CodeEvaluator{
				EvaluatorID: "valid-result",
				MinScore:    -1,
				MaxScore:    2,
				Fn: func(SimulationResult) (EvaluationResult, error) {
					return EvaluationResult{Score: 2, Feedback: "  preserved  "}, nil
				},
			},
		},
	}

	result, err := RunEnvironment(context.Background(), t.TempDir(), environment, echoAdapter{})
	if err != nil {
		t.Fatal(err)
	}
	if invalidInvoked {
		t.Fatal("malformed evaluator was invoked")
	}
	if len(result.EvaluationErrors) != 2 {
		t.Fatalf("expected two evaluation errors, got %#v", result.EvaluationErrors)
	}
	if len(result.Evaluations) != 1 {
		t.Fatalf("expected later evaluator to continue, got %#v", result.Evaluations)
	}
	evaluation := result.Evaluations[0]
	if evaluation.EvaluatorID != "valid-result" || evaluation.Score != 2 || evaluation.Feedback != "  preserved  " || evaluation.MinScore != -1 || evaluation.MaxScore != 2 {
		t.Fatalf("unexpected successful evaluation: %#v", evaluation)
	}
	serialized, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(serialized), `"min_score":-1,"max_score":2`) {
		t.Fatalf("resolved result bounds were not serialized: %s", serialized)
	}
}

func TestRunEnvironmentAcceptsFixedInputPointer(t *testing.T) {
	environment := Environment{
		ID:          "support-pointer-smoke",
		Name:        "Support pointer smoke",
		Description: "Checks a fixed input pointer.",
		Target:      AgentTarget{},
		Input:       &FixedInput{Turns: []FixedTurn{{Content: "hello"}}},
	}

	result, err := RunEnvironment(context.Background(), t.TempDir(), environment, echoAdapter{})
	if err != nil {
		t.Fatal(err)
	}
	if result.FinalOutput != "hello" {
		t.Fatalf("unexpected final output: %#v", result.FinalOutput)
	}
}

func TestRunEnvironmentExecutesComponentTarget(t *testing.T) {
	environment := Environment{
		ID:          "lookup-smoke",
		Name:        "Lookup smoke",
		Description: "Checks one component call.",
		Target:      ComponentTarget{ImportPath: "tools:lookup"},
		Input: FixedComponentInput{
			Args:      []JSONValue{"weather"},
			NamedArgs: map[string]JSONValue{"city": "Paris"},
		},
	}

	result, err := RunComponentEnvironment(context.Background(), t.TempDir(), environment, componentEchoAdapter{})
	if err != nil {
		t.Fatal(err)
	}
	output, ok := result.FinalOutput.(map[string]JSONValue)
	if !ok || output["target"] != "tools:lookup" || output["city"] != "Paris" {
		t.Fatalf("unexpected component output: %#v", result.FinalOutput)
	}
	if result.EventsCount != 4 {
		t.Fatalf("unexpected events count: %d", result.EventsCount)
	}
	if result.Metadata["target"] != "component" {
		t.Fatalf("unexpected target metadata: %#v", result.Metadata)
	}
	if result.Timings == nil || len(result.Timings.ComponentRuns) != 1 {
		t.Fatalf("unexpected component timings: %#v", result.Timings)
	}
	componentRun := result.Timings.ComponentRuns[0]
	if componentRun.Target != "tools:lookup" || componentRun.DurationMS < 0 || result.Timings.TotalDurationMS != componentRun.DurationMS {
		t.Fatalf("unexpected component timing: %#v", result.Timings)
	}
}

func TestRunEnvironmentExecutesComponentTargetPointer(t *testing.T) {
	environment := Environment{
		ID:          "lookup-pointer-smoke",
		Name:        "Lookup pointer smoke",
		Description: "Checks a component target pointer.",
		Target:      &ComponentTarget{ImportPath: "tools:lookup"},
		Input:       &FixedComponentInput{Args: []JSONValue{"weather"}},
	}

	result, err := RunComponentEnvironment(context.Background(), t.TempDir(), environment, componentEchoAdapter{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Metadata["target"] != "component" {
		t.Fatalf("unexpected target metadata: %#v", result.Metadata)
	}
}

func TestRunEnvironmentGeneratesComponentInput(t *testing.T) {
	environment := Environment{
		ID:          "generated-lookup-smoke",
		Name:        "Generated lookup smoke",
		Description: "Checks generated component input.",
		Target:      ComponentTarget{ImportPath: "tools:lookup"},
		Input: &GeneratedComponentInput{
			Instructions: "Look up Paris weather.",
			Model:        ModelSpec{Name: "test-model"},
			InputSchema:  map[string]JSONValue{"type": "object"},
		},
	}

	result, err := runComponentEnvironmentWithModelFactory(context.Background(), t.TempDir(), environment, componentEchoAdapter{}, nil, func(context.Context, ModelSpec) (adkmodel.LLM, error) {
		return staticJudgeModel{response: `{"args":["weather"],"kwargs":{"city":"Paris"}}`, expectedInstruction: "Return JSON only with args"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	output := result.FinalOutput.(map[string]JSONValue)
	if output["arg"] != "weather" || output["city"] != "Paris" {
		t.Fatalf("unexpected generated component output: %#v", output)
	}
}

func TestRunEnvironmentGeneratesPersonaInput(t *testing.T) {
	environment := Environment{
		ID:          "persona-smoke",
		Name:        "Persona smoke",
		Description: "Checks a generated user turn.",
		Target:      AgentTarget{},
		Input: &PersonaInput{
			Persona:  "A traveler asking about weather.",
			MaxTurns: 1,
			Model:    ModelSpec{Name: "test-model"},
		},
	}

	result, err := runAgentEnvironmentWithModelFactory(context.Background(), t.TempDir(), environment, echoAdapter{}, nil, func(context.Context, ModelSpec) (adkmodel.LLM, error) {
		return staticJudgeModel{response: `{"content":"What is the weather in Paris?","should_stop":false}`, expectedInstruction: "Impersonate the specified user persona"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.FinalOutput != "What is the weather in Paris?" {
		t.Fatalf("unexpected persona output: %#v", result.FinalOutput)
	}
	if result.StopReason != "persona complete" {
		t.Fatalf("unexpected stop reason: %q", result.StopReason)
	}
	serialized, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(serialized), `"component_runs":[]`) {
		t.Fatalf("expected empty component runs array, got %s", serialized)
	}
}

func TestRunEnvironmentRejectsNonPositivePersonaMaxTurns(t *testing.T) {
	environment := Environment{
		ID:          "invalid-persona",
		Name:        "Invalid persona",
		Description: "Checks persona validation.",
		Target:      AgentTarget{},
		Input:       &PersonaInput{Persona: "A user", MaxTurns: 0},
	}

	_, err := RunEnvironment(context.Background(), t.TempDir(), environment, echoAdapter{})
	if err == nil || !strings.Contains(err.Error(), "max turns must be greater than zero") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunEnvironmentSerializesEvaluatorFailuresWithMessage(t *testing.T) {
	environment := Environment{
		ID:          "support-smoke",
		Name:        "Support smoke",
		Description: "Checks one support turn.",
		Target:      AgentTarget{},
		Input:       FixedInput{Turns: []FixedTurn{{Content: "hello"}}},
		Evaluators: []Evaluator{
			CodeEvaluator{
				EvaluatorID: "failing-evaluator",
				Description: "Fails deliberately.",
				Fn: func(SimulationResult) (EvaluationResult, error) {
					return EvaluationResult{}, errors.New("judge unavailable")
				},
			},
		},
	}

	result, err := RunEnvironment(context.Background(), t.TempDir(), environment, echoAdapter{})
	if err != nil {
		t.Fatal(err)
	}
	serialized, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	jsonText := string(serialized)
	if !strings.Contains(jsonText, `"message":"judge unavailable"`) {
		t.Fatalf("expected evaluator failure to serialize as message, got %s", jsonText)
	}
	if strings.Contains(jsonText, `"error":"judge unavailable"`) {
		t.Fatalf("evaluator failure used obsolete error field: %s", jsonText)
	}
}

func TestRunEnvironmentExecutesScopeMatchedGlobalEvaluator(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".relai"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(projectRoot, ".relai", "learning-env-context.json"),
		[]byte(`{"components":[{"id":"lookup","import_path":"tools:lookup"}]}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	globalEvaluator := CodeEvaluator{
		EvaluatorID: "global-quality",
		Description: "Checks every agent simulation.",
		Scope:       "end-to-end",
		Fn: func(SimulationResult) (EvaluationResult, error) {
			return EvaluationResult{Score: 1, Feedback: "Global check passed."}, nil
		},
	}
	environment := Environment{
		ID:          "support-smoke",
		Name:        "Support smoke",
		Description: "Checks one support turn.",
		Target:      AgentTarget{},
		Input:       FixedInput{Turns: []FixedTurn{{Content: "hello"}}},
	}
	matched, err := FilterGlobalEvaluatorsForEnvironment(projectRoot, []Evaluator{globalEvaluator}, environment)
	if err != nil {
		t.Fatal(err)
	}
	environment.Evaluators, err = CombineEvaluators(environment.Evaluators, matched)
	if err != nil {
		t.Fatal(err)
	}
	result, err := RunEnvironment(context.Background(), projectRoot, environment, echoAdapter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Evaluations) != 1 || !result.Evaluations[0].IsGlobalEvaluator {
		t.Fatalf("expected one marked global evaluation, got %#v", result.Evaluations)
	}
}

func TestFilterGlobalEvaluatorsUsesComponentScope(t *testing.T) {
	projectRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectRoot, ".relai"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(projectRoot, ".relai", "learning-env-context.json"),
		[]byte(`{"components":[{"id":"lookup","import_path":"tools:lookup"}]}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	environment := Environment{
		Target: ComponentTarget{ImportPath: "tools:lookup"},
		Input:  FixedComponentInput{Args: []JSONValue{"weather"}},
	}
	matched, err := FilterGlobalEvaluatorsForEnvironment(projectRoot, []Evaluator{
		CodeEvaluator{EvaluatorID: "lookup-check", Scope: "lookup"},
		CodeEvaluator{EvaluatorID: "agent-check", Scope: "end-to-end"},
	}, environment)
	if err != nil {
		t.Fatal(err)
	}
	if len(matched) != 1 || matched[0].ID() != "lookup-check" {
		t.Fatalf("unexpected matched evaluators: %#v", matched)
	}
}

func TestLLMJudgeEvaluatorRunsThroughADK(t *testing.T) {
	evaluator := LLMJudgeEvaluator{
		EvaluatorID:  "off-topic",
		Instructions: "Refuse off-topic requests.",
		Model:        ModelSpec{Name: "test-model"},
	}
	result, err := runLLMJudge(context.Background(), evaluator, SimulationResult{
		FinalOutput: "I can only help with finance questions.",
		Metadata:    map[string]any{"target": "agent"},
	}, func(context.Context, ModelSpec) (adkmodel.LLM, error) {
		return staticJudgeModel{response: `{"score":1,"feedback":"Correct refusal."}`, expectedInstruction: "Refuse off-topic requests"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Score != 1 || result.Feedback != "Correct refusal." {
		t.Fatalf("unexpected judge result: %#v", result)
	}
}

func TestLLMJudgeRejectsMalformedBoundsBeforeModelInvocation(t *testing.T) {
	invoked := false
	_, err := runLLMJudge(context.Background(), LLMJudgeEvaluator{
		EvaluatorID: "invalid-judge",
		MinScore:    1,
		MaxScore:    1,
	}, SimulationResult{}, func(context.Context, ModelSpec) (adkmodel.LLM, error) {
		invoked = true
		return staticJudgeModel{response: `{"score":1,"feedback":"ok"}`}, nil
	})
	if err == nil {
		t.Fatal("expected malformed judge bounds to fail")
	}
	if invoked {
		t.Fatal("judge model was invoked despite malformed bounds")
	}
}

func TestSmokeTestEvaluatorValidatesBoundsAndResults(t *testing.T) {
	for _, evaluator := range []Evaluator{
		CodeEvaluator{
			EvaluatorID: "invalid-bounds",
			MinScore:    2,
			MaxScore:    1,
			Fn: func(SimulationResult) (EvaluationResult, error) {
				return EvaluationResult{Score: 1, Feedback: "ok"}, nil
			},
		},
		CodeEvaluator{
			EvaluatorID: "invalid-result",
			Fn: func(SimulationResult) (EvaluationResult, error) {
				return EvaluationResult{Score: 1, Feedback: "  "}, nil
			},
		},
	} {
		if err := SmokeTestEvaluator(evaluator); err == nil {
			t.Fatalf("expected smoke validation to reject %q", evaluator.ID())
		}
	}
}

func TestOpenAICompatibleJudgeModelUsesProviderConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected request path: %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", request.Header.Get("Authorization"))
		}
		var payload openAIChatRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Model != "judge-model" || payload.ResponseFormat == nil || payload.ResponseFormat.Type != "json_object" {
			t.Fatalf("unexpected provider payload: %#v", payload)
		}
		if len(payload.Messages) != 2 || payload.Messages[0].Role != "system" || payload.Messages[1].Role != "user" {
			t.Fatalf("unexpected provider messages: %#v", payload.Messages)
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"score\":1,\"feedback\":\"ok\"}"}}]}`))
	}))
	defer server.Close()
	t.Setenv("RELAI_MODEL_PROVIDER_TEST_PROVIDER_API_KEY", "test-key")
	t.Setenv("RELAI_MODEL_PROVIDER_TEST_PROVIDER_BASE_URL", server.URL)

	configured, err := newOpenAICompatibleModel(ModelSpec{Name: "judge-model", Provider: "test-provider"})
	if err != nil {
		t.Fatal(err)
	}
	response, err := nextLLMResponse(configured.GenerateContent(context.Background(), &adkmodel.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("evaluate", genai.RoleUser)},
		Config:   &genai.GenerateContentConfig{SystemInstruction: genai.NewContentFromText("judge", genai.RoleUser)},
	}, false))
	if err != nil {
		t.Fatal(err)
	}
	if contentText(response.Content) != `{"score":1,"feedback":"ok"}` {
		t.Fatalf("unexpected provider response: %#v", response)
	}
}

func nextLLMResponse(sequence iter.Seq2[*adkmodel.LLMResponse, error]) (*adkmodel.LLMResponse, error) {
	for response, err := range sequence {
		return response, err
	}
	return nil, errors.New("model returned no response")
}
