# Airline integration smoke

`environment/scenario.json` drives a two-turn booking lookup and contextual seat change. `tests/evaluate.py` checks real `lookup_booking` and `change_seat` calls, arguments and same-ID results using only the exported ATIF trajectory. The separate verifier has no project mount or provider credentials. `instruction.md` is intentionally empty and required by Harbor.

Editable task behavior belongs in `task.toml`, `environment/scenario.json`, optional `environment/mocks/`, and declared evaluators under `tests/`. There are no mocks in this task: airline tools use the repository's real in-memory demo bookings.

The reusable leaf adapter lives under `.relai/harbor/adapter/`, with an explicit staged snapshot in this task. It compiles the current `/workspace` Go package to a unique `/tmp` binary from the module root, retains one Agent, and replays the supplied message history each turn. The project owns recording at its actual provider/tool boundaries. Synchronous mutex-protected snapshots leave no pending writer work on response, failure, or cancellation.

Configure at least one provider key privately in `.relai/simulator.env` (or the process environment). OpenAI is preferred if both keys are present. Model overrides retain the repository's defaults. Leave `RELAI_DEMO_CHALLENGE` empty for normal booking tools. Never store credential values in task assets.

Packaging owns `environment/Dockerfile`, `tests/test.sh`, `generated/build.json`, and `generated/dependencies.json`. Reconfigure through `configure_harbor_packaging`; do not hand-edit generated files. `generated/runtime/` retains exact shared runtime inputs. Go 1.24 dependencies are installed with `go mod download` and verified against the staged `go.mod`/`go.sum`; no project source is copied into the image. Python is installed explicitly for the launcher. The preflight Go import probe resolves the selected factory under `/workspace` without provider calls.
