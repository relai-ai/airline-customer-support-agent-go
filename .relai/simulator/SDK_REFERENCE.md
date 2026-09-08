# RELAI Go Simulator SDK Reference

Use this reference instead of reading RELAI SDK source while scaffolding a Go
project simulator.

## Learning Environments

- Go learning environment files use `package main`.
- A learning environment exports `func BuildEnvironment() relai.Environment`.
- Agent environments use `relai.FixedInput` or `relai.PersonaInput`. Fixed input supports multiple turns; persona input generates up to `MaxTurns` user turns using its configured model.
- Component environments use `relai.ComponentTarget{ImportPath: "module/path:Export"}` with `relai.FixedComponentInput` or `relai.GeneratedComponentInput`.
- Every `relai.Environment`, `relai.CodeEvaluator`, and `relai.LLMJudgeEvaluator` needs a short `Description`.
- Code evaluators read the final assistant response from
  `relai.SimulationResult.FinalOutput`. The Go result does not expose a
  `Turns` field; use `Events` only for transcript-event assertions.
- Omit both `MinScore` and `MaxScore` to use the default inclusive `0..1`
  range. Custom bounds must be finite and satisfy `MinScore < MaxScore`; an
  explicit `0..0` range cannot be represented because the zero pair means
  omitted bounds.
- Evaluator scores must be finite and inclusively within the resolved bounds.
  Feedback must contain non-whitespace text and is preserved exactly when
  valid. Invalid evaluator output is recorded in `EvaluationErrors`, and later
  evaluators still run. Successful `EvaluationEntry` values include the
  resolved `MinScore` and `MaxScore`.
- `relai.SimulationResult.Timings` records `TotalDurationMS` and ordered
  `ComponentRuns`; it excludes input generation and evaluator execution.

## Global Evaluators

- Go global evaluator files live in `.relai/evaluators/*.go` and use an importable package name such as `package evaluator`; do not use `package main`.
- Each file exports `func BuildEvaluator() relai.Evaluator`.
- RELAI isolates each evaluator file into its own package at simulation time, so multiple global evaluators may use the same exported function name.
- A global evaluator must declare `Scope: "end-to-end"` or a component ID from `.relai/learning-env-context.json`. RELAI runs only evaluators whose scope matches the selected learning environment.

## Runner And Adapter Contract

Implement `.relai/simulator/src/relai_simulator/adapter.go`. The runner creates
`relai.AdapterRuntime` from the learning environment's named tool overrides and
passes it to `BuildAgentAdapter(runtime)`.

For agent environments, `BuildAgentAdapter(runtime)` must return a value implementing:

```go
RunTurn(ctx context.Context, input relai.JSONValue) (relai.AgentTurnResult, error)
```

It must also implement `AdapterCapabilities()` and declare `run_turn`. Optional
capabilities are `named_targets`, `tool_overrides`, `tool_events`, and
`run_component`. An adapter declaring any optional capability must implement
`SelfCheck(ctx, runtime)` and behaviorally verify it with local sentinels or
RELAI overrides, without provider calls, credentials, or project mutations.
Framework conversion and replacement mechanics belong in this generated adapter. Prefer injecting
`runtime.ToolOverrides` while constructing the project agent and use
`runtime.RecordMockCall` from translated wrappers.

Use `inspect_dependency` to resolve Go packages through `go list` and inspect
bounded declarations or source for the exact module version. Do not infer
integration behavior from a framework name.

For a shared runtime with named logical agents, also implement:

```go
ConfigureAgentTarget(ctx context.Context, target relai.AgentTarget) error
```

The runner calls this before the first turn when an environment uses
`relai.AgentTarget{AgentTarget: "support"}`. The adapter can use the selector to
construct or configure the corresponding runtime. Existing default-agent
adapters do not need to implement this interface.

For component environments, the same adapter may additionally implement:

```go
RunComponent(ctx context.Context, target relai.ComponentTarget, input relai.FixedComponentInput) (relai.JSONValue, error)
```

`target.ImportPath` identifies the component selected by the learning environment.
`input.Args` and `input.NamedArgs` contain the fixed positional and named
arguments. The serialized JSON field remains `kwargs` for cross-SDK compatibility.

`PersonaInput` and `GeneratedComponentInput` use the same model provider
configuration as `LLMJudgeEvaluator`. Their models must return JSON; the SDK
requests this automatically. A persona model returns `content`, `should_stop`,
and optional `reason`. A generated component-input model returns `args` and
`kwargs`.

Import `context` when implementing this method. `input` is the JSON-serializable fixed-turn content value. It may be a string,
object, array, number, boolean, or nil.

## Init Smoke Validation

Write `.relai/simulator/smoke_learning_env.go` as a minimal
`relai.Environment` that runs one representative, project-valid fixed turn
through the generated adapter. Use the same public input shape that normal
learning environments should use.

## Dependency Edits

Do not manually add RELAI SDK release or local-path logic. The CLI owns RELAI
SDK installation in `.relai/go.mod` so simulator files and
`.relai/learning-envs/*.go` can import `github.com/relai-ai/relai-cli/sdk/go`.
If the project needs extra RELAI-local dependencies, edit only the
`BEGIN PROJECT DEPENDENCY INSTALL` section of `.relai/simulator/install.sh`.
