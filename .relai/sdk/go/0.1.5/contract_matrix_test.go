package relai

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type semanticContractMatrix struct {
	SchemaVersion          string                         `json:"schema_version"`
	SimulationResults      []simulationResultContractCase `json:"simulation_results"`
	ComponentFailures      []componentFailureContractCase `json:"component_failures"`
	EvaluatorCases         []evaluatorContractCase        `json:"evaluator_cases"`
	BenchmarkSampleIDCases []benchmarkIDContractCase      `json:"benchmark_sample_id_cases"`
}

type simulationResultContractCase struct {
	ID          string          `json:"id"`
	Payload     json.RawMessage `json:"payload"`
	SelectedRun struct {
		LearningEnvironmentID string `json:"learning_environment_id"`
		RunID                 string `json:"run_id"`
		FinalOutputIsNull     bool   `json:"final_output_is_null"`
	} `json:"selected_run"`
	ParseBehavior map[string]string `json:"parse_behavior"`
}

type componentFailureContractCase struct {
	ID       string `json:"id"`
	Target   string `json:"target"`
	Message  string `json:"message"`
	Expected struct {
		StopReason        string `json:"stop_reason"`
		FinalOutputIsNull bool   `json:"final_output_is_null"`
		EventType         string `json:"event_type"`
		EvaluatorsRun     bool   `json:"evaluators_run"`
	} `json:"expected"`
}

type evaluatorContractCase struct {
	ID          string  `json:"id"`
	MinScore    float64 `json:"min_score"`
	MaxScore    float64 `json:"max_score"`
	Score       float64 `json:"score"`
	Feedback    string  `json:"feedback"`
	BoundsValid bool    `json:"bounds_valid"`
	ResultValid bool    `json:"result_valid"`
}

type benchmarkIDContractCase struct {
	ID             string `json:"id"`
	CSV            string `json:"csv"`
	SampleIDColumn string `json:"sample_id_column"`
	Expected       struct {
		Valid     bool     `json:"valid"`
		SampleIDs []string `json:"sample_ids"`
	} `json:"expected"`
}

type contractFailingComponentAdapter struct {
	message string
}

func (adapter contractFailingComponentAdapter) RunComponent(
	context.Context,
	ComponentTarget,
	FixedComponentInput,
) (JSONValue, error) {
	return nil, errors.New(adapter.message)
}

func loadSemanticContractMatrix(t *testing.T) semanticContractMatrix {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("..", "..", "contract", "fixtures", "semantic_contract_matrix.json"))
	if err != nil {
		t.Fatalf("read semantic contract matrix: %v", err)
	}
	var matrix semanticContractMatrix
	if err := json.Unmarshal(contents, &matrix); err != nil {
		t.Fatalf("parse semantic contract matrix: %v", err)
	}
	if matrix.SchemaVersion != "relai.semantic_contract_matrix.v1" {
		t.Fatalf("unexpected semantic contract matrix schema: %q", matrix.SchemaVersion)
	}
	return matrix
}

func TestContractMatrixSimulationResults(t *testing.T) {
	matrix := loadSemanticContractMatrix(t)
	for _, testCase := range matrix.SimulationResults {
		t.Run(testCase.ID, func(t *testing.T) {
			behavior := testCase.ParseBehavior["go"]
			if behavior == "not_exposed" {
				return
			}
			if behavior != "accept" && behavior != "ignore_additive" {
				t.Fatalf("unknown Go parse behavior %q", behavior)
			}

			var result SimulationResult
			if err := json.Unmarshal(testCase.Payload, &result); err != nil {
				t.Fatalf("unmarshal canonical direct result: %v", err)
			}
			if result.LearningEnvironmentID != testCase.SelectedRun.LearningEnvironmentID {
				t.Fatalf("learning environment ID = %q, want %q", result.LearningEnvironmentID, testCase.SelectedRun.LearningEnvironmentID)
			}
			if result.RunID != testCase.SelectedRun.RunID {
				t.Fatalf("run ID = %q, want %q", result.RunID, testCase.SelectedRun.RunID)
			}
			if (result.FinalOutput == nil) != testCase.SelectedRun.FinalOutputIsNull {
				t.Fatalf("final output null meaning differs: %#v", result.FinalOutput)
			}
		})
	}
}

func TestContractMatrixComponentFailures(t *testing.T) {
	matrix := loadSemanticContractMatrix(t)
	for _, testCase := range matrix.ComponentFailures {
		t.Run(testCase.ID, func(t *testing.T) {
			evaluatorRan := false
			environment := Environment{
				ID:          "contract-component-failure",
				Name:        "Contract component failure",
				Description: "Exercises canonical component failure semantics.",
				Target:      ComponentTarget{ImportPath: testCase.Target},
				Input:       FixedComponentInput{Args: []JSONValue{}},
				Evaluators: []Evaluator{CodeEvaluator{
					EvaluatorID: "saw-error",
					Description: "Checks the canonical component error event.",
					Fn: func(result SimulationResult) (EvaluationResult, error) {
						evaluatorRan = true
						for _, event := range result.Events {
							if event.EventType == testCase.Expected.EventType {
								return EvaluationResult{Score: 1, Feedback: "captured"}, nil
							}
						}
						return EvaluationResult{Score: 0, Feedback: "missing"}, nil
					},
				}},
			}

			result, err := RunComponentEnvironment(
				context.Background(),
				t.TempDir(),
				environment,
				contractFailingComponentAdapter{message: testCase.Message},
			)
			if err != nil {
				t.Fatalf("component invocation failure returned an operational error: %v", err)
			}
			if result.StopReason != testCase.Expected.StopReason {
				t.Fatalf("stop reason = %q, want %q", result.StopReason, testCase.Expected.StopReason)
			}
			if (result.FinalOutput == nil) != testCase.Expected.FinalOutputIsNull {
				t.Fatalf("final output null meaning differs: %#v", result.FinalOutput)
			}
			componentError, ok := result.Metadata["component_error"].(map[string]any)
			if !ok || componentError["message"] != testCase.Message {
				t.Fatalf("unexpected component failure metadata: %#v", result.Metadata)
			}
			if evaluatorRan != testCase.Expected.EvaluatorsRun {
				t.Fatalf("evaluator ran = %t, want %t", evaluatorRan, testCase.Expected.EvaluatorsRun)
			}
			if len(result.Evaluations) != 1 || result.Evaluations[0].Feedback != "captured" {
				t.Fatalf("unexpected component failure evaluations: %#v", result.Evaluations)
			}
		})
	}
}

func TestContractMatrixEvaluatorBoundsAndResults(t *testing.T) {
	matrix := loadSemanticContractMatrix(t)
	for _, testCase := range matrix.EvaluatorCases {
		t.Run(testCase.ID, func(t *testing.T) {
			evaluator := CodeEvaluator{
				EvaluatorID: testCase.ID,
				Description: "Exercises canonical evaluator semantics.",
				MinScore:    testCase.MinScore,
				MaxScore:    testCase.MaxScore,
				Fn: func(SimulationResult) (EvaluationResult, error) {
					return EvaluationResult{Score: testCase.Score, Feedback: testCase.Feedback}, nil
				},
			}
			result, err := evaluator.Evaluate(SimulationResult{})
			if !testCase.BoundsValid || !testCase.ResultValid {
				if err == nil {
					t.Fatalf("expected invalid evaluator case, got result %#v", result)
				}
				return
			}
			if err != nil {
				t.Fatalf("evaluate canonical valid case: %v", err)
			}
			if result.Score != testCase.Score {
				t.Fatalf("score = %v, want %v", result.Score, testCase.Score)
			}
		})
	}
}

func TestContractMatrixBenchmarkSampleIDs(t *testing.T) {
	matrix := loadSemanticContractMatrix(t)
	for _, testCase := range matrix.BenchmarkSampleIDCases {
		t.Run(testCase.ID, func(t *testing.T) {
			csvPath := filepath.Join(t.TempDir(), testCase.ID+".csv")
			if err := os.WriteFile(csvPath, []byte(testCase.CSV), 0o600); err != nil {
				t.Fatalf("write canonical benchmark CSV: %v", err)
			}
			benchmark := Benchmark{
				ID:              "contract-" + testCase.ID,
				Name:            "Contract benchmark",
				Description:     "Exercises canonical benchmark sample identity.",
				RequiredColumns: []string{testCase.SampleIDColumn, "prompt"},
				SampleIDColumn:  testCase.SampleIDColumn,
				DatasetRef:      LocalBenchmarkCSV(csvPath),
			}
			expected := testCase.Expected
			samples, err := ListBenchmarkSamples(benchmark, "", "")
			if !expected.Valid {
				if err == nil {
					t.Fatalf("expected invalid sample IDs, got %#v", samples)
				}
				return
			}
			if err != nil {
				t.Fatalf("list canonical benchmark samples: %v", err)
			}
			if len(samples) != len(expected.SampleIDs) {
				t.Fatalf("sample count = %d, want %d", len(samples), len(expected.SampleIDs))
			}
			for index, sample := range samples {
				if sample.ID != expected.SampleIDs[index] {
					t.Fatalf("sample %d ID = %q, want %q", index, sample.ID, expected.SampleIDs[index])
				}
			}
		})
	}
}
