#!/usr/bin/env python3
"""Task-independent persistent NDJSON launcher for ATIF-instrumented adapters."""
import argparse
import codecs
import json
import os
import shlex
import signal
import subprocess
import sys
import threading
import urllib.error
import urllib.request
from pathlib import Path

# The CLI's host relay for persona turns (see src/harbor/relay.rs).
MODEL_RELAY_URL = "RELAI_MODEL_RELAY_URL"
MODEL_RELAY_TOKEN = "RELAI_MODEL_RELAY_TOKEN"
USER_INPUT_STEP_MODE = "RELAI_HARBOR_USER_INPUT_STEP_MODE"
USER_INPUT_STEP_WARNING_FILE = "relai_launcher_warnings.jsonl"
# Head-and-tail diagnostic size; must match DIAGNOSTIC_BYTES in src/harbor/mod.rs.
DIAGNOSTIC_LIMIT = 16 * 1024


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--scenario", type=Path, required=True)
    parser.add_argument("--trajectory", type=Path, required=True)
    parser.add_argument("--agent-command", required=True)
    parser.add_argument("--descriptor", type=Path)
    args = parser.parse_args()
    scenario = json.loads(args.scenario.read_text(encoding="utf-8"))
    descriptor = load_descriptor(args.descriptor)
    scenario, descriptor = apply_descriptor(scenario, descriptor)
    turns, persona_spec, services = validate_scenario(scenario)
    history = []
    previous_steps = None
    user_input_step_mode = os.environ.get(USER_INPUT_STEP_MODE, "strict")
    if user_input_step_mode not in ("strict", "warn"):
        raise ValueError(f"{USER_INPUT_STEP_MODE} must be 'strict' or 'warn'")
    command, adapter_cwd = task_adapter_command(descriptor, args.agent_command)
    child_env = dict(os.environ)
    # The relay token is the launcher's alone; the adapter never sees it.
    child_env.pop(MODEL_RELAY_URL, None)
    child_env.pop(MODEL_RELAY_TOKEN, None)
    child_env["RELAI_ATIF_TRAJECTORY_PATH"] = str(args.trajectory)
    adapter = subprocess.Popen(
        command,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        # Keep the protocol pipes binary so the launcher can drain stderr and
        # inspect trailing stdout without a buffered TextIO wrapper blocking on
        # a descriptor held by an adapter descendant.
        text=False,
        env=child_env,
        cwd=adapter_cwd,
        # A trial owns the entire adapter session. This lets cleanup terminate
        # a forked helper before it can retain a protocol pipe or outlive the
        # Harbor trial.
        start_new_session=True,
    )
    redactions = redaction_variants(runtime_values(child_env))
    stderr_tail = StderrTail(adapter.stderr, redactions=redactions)
    try:
      for index, turn in enumerate(turns):
        if persona_spec is None:
            content = turn
        elif index == 0 and isinstance(persona_spec.get("seed_message"), str):
            content = persona_spec["seed_message"]
        else:
            generated = generate_persona_turn(persona_spec, history)
            if generated.get("should_stop"):
                break
            content = generated.get("content")
            if not isinstance(content, str) or not content:
                raise ValueError("persona model response requires non-empty string content")
        request = {
            "scenario_id": scenario.get("id"), "turn_index": index, "user_input": content,
            "services": services, "history": history,
            "task": descriptor,
        }
        try:
            adapter.stdin.write((json.dumps(request, separators=(",", ":")) + "\n").encode())
            adapter.stdin.flush()
        except BrokenPipeError as error:
            raise adapter_failure(adapter, stderr_tail, "adapter closed its request stream") from error
        framed_response = read_framed_response(adapter)
        if not framed_response:
            raise adapter_failure(adapter, stderr_tail, "adapter closed stdout before its framed response")
        try:
            response = json.loads(framed_response.decode())
        except json.JSONDecodeError as error:
            raise RuntimeError(
                "adapter stdout must contain exactly one JSON response line per request; "
                "received malformed output: "
                f"{bounded_diagnostic(redact(framed_response.decode(errors='replace').strip(), redactions))!r}"
            ) from error
        if not isinstance(response, dict):
            raise ValueError("agent response must be a JSON object")
        history.append({"role": "user", "content": content})
        assistant_message = response.get("assistant_message")
        if not isinstance(assistant_message, str):
            raise ValueError("agent response must be an object with assistant_message")
        history.append({"role": "assistant", "content": assistant_message})
        try:
            current_steps = trajectory_steps(args.trajectory)
            if previous_steps is not None:
                if len(current_steps) <= len(previous_steps) or current_steps[:len(previous_steps)] != previous_steps:
                    raise RuntimeError(
                        "project adapter replaced prior ATIF trajectory steps instead of preserving "
                        "the cumulative trajectory; repair its instrumentation to retain the prior prefix."
                    )
                new_steps = current_steps[len(previous_steps):]
            else:
                new_steps = current_steps
            if not has_matching_user_step(new_steps, content):
                message = user_input_step_message(index, content)
                if user_input_step_mode == "warn":
                    emit_user_input_step_warning(args.trajectory, message)
                else:
                    raise RuntimeError(message)
            if not any(isinstance(step, dict) and step.get("source") == "agent" for step in new_steps):
                raise RuntimeError(
                    "project adapter returned a framed response without recording a new source=agent "
                    "ATIF step for that turn; repair its instrumentation at the observed agent boundary."
                )
        except RuntimeError as error:
            raise RuntimeError(f"{error}\n{turn_evidence(assistant_message, stderr_tail)}") from None
        previous_steps = current_steps
    finally:
        shutdown_adapter(adapter, stderr_tail, redactions)


class StderrTail:
    """Continuously drain adapter stderr while retaining a bounded, redacted
    diagnostic that keeps both its beginning and its end."""

    def __init__(self, stream, limit=DIAGNOSTIC_LIMIT, redactions=()):
        self.limit = limit
        self.redactions = redactions
        self.decoder = codecs.getincrementaldecoder("utf-8")(errors="replace")
        # Unredacted text that may still become part of a forwarded value.
        self.pending = ""
        self.retained = ""
        self.lock = threading.Lock()
        self.thread = threading.Thread(target=self._drain, args=(stream,), daemon=True)
        self.thread.start()

    def _drain(self, stream):
        try:
            while True:
                chunk = os.read(stream.fileno(), 4096)
                text = self.pending + self.decoder.decode(chunk, final=not chunk)
                split = self._settled(text) if chunk else len(text)
                self.pending = text[split:]
                with self.lock:
                    self.retained = bounded_diagnostic(
                        self.retained + redact(text[:split], self.redactions), self.limit)
                if not chunk:
                    return
        except OSError:
            # Cleanup can close the pipe after the process group is gone.
            pass

    def _settled(self, text):
        """Length of the prefix that no forwarded value can still extend into:
        stop before any value left incomplete at the end, and never cut a
        complete value in two."""
        split = len(text)
        for variant in self.redactions:
            for start in range(max(0, len(text) - len(variant) + 1), split):
                if variant.startswith(text[start:]):
                    split = start
                    break
        moved = True
        while moved:
            moved = False
            for variant in self.redactions:
                start = text.find(variant, max(0, split - len(variant) + 1))
                if -1 < start < split:
                    split, moved = start, True
        return split

    def text(self):
        with self.lock:
            return self.retained

    def close(self):
        self.thread.join(timeout=1)


def bounded_diagnostic(text, limit=DIAGNOSTIC_LIMIT):
    """Keep the initial cause and the final status within `limit` characters."""
    marker = "\n... [truncated] ...\n"
    if len(text) <= limit:
        return text
    kept = max(0, limit - len(marker))
    return text[:kept // 2] + marker + text[len(text) - (kept - kept // 2):]


def has_matching_user_step(steps, content):
    return any(
        isinstance(step, dict)
        and step.get("source") == "user"
        and isinstance(step.get("message"), str)
        and normalized_expected_input(content) in normalized_expected_input(step["message"])
        for step in steps
    )


def normalized_expected_input(text):
    return " ".join(str(text).split())


def preview_text(text, limit=160):
    normalized = normalized_expected_input(text)
    if len(normalized) <= limit:
        return normalized
    return normalized[:limit // 2] + " ... " + normalized[-(limit - limit // 2 - 5):]


def user_input_step_message(turn_index, content):
    return (
        "project adapter returned a framed response without recording a new source=user "
        "ATIF step containing that turn's user_input; repair its instrumentation at the "
        f"observed user boundary. turn_index={turn_index} "
        f"expected_input_preview={preview_text(content)!r}"
    )


def emit_user_input_step_warning(trajectory, message):
    diagnostic = "RELAI_HARBOR_USER_INPUT_STEP_WARNING: " + message
    print(diagnostic, file=sys.stderr, flush=True)
    warning = {"code": "missing_user_input_step", "message": message}
    with (trajectory.parent / USER_INPUT_STEP_WARNING_FILE).open("a", encoding="utf-8") as file:
        file.write(json.dumps(warning, separators=(",", ":")) + "\n")


def runtime_values(environment):
    """Values forwarded into the agent phase, named in RELAI_HARBOR_RUNTIME_ENV_NAMES."""
    names = environment.get("RELAI_HARBOR_RUNTIME_ENV_NAMES", "").split(",")
    return [environment[name] for name in names if name and environment.get(name)]


def redaction_variants(values):
    variants = {variant for value in values for variant in (value, json.dumps(value)[1:-1])}
    return tuple(sorted(variants, key=len, reverse=True))


def redact(text, variants):
    for variant in variants:
        text = text.replace(variant, "[redacted]")
    return text


def read_framed_response(adapter):
    """Read one NDJSON response without sharing stdout with a text buffer."""
    import select
    buffer = getattr(adapter, "_relai_stdout_buffer", bytearray())
    while True:
        newline = buffer.find(b"\n")
        if newline >= 0:
            adapter._relai_stdout_buffer = buffer[newline + 1:]
            return bytes(buffer[:newline + 1])
        readable, _, _ = select.select([adapter.stdout], [], [])
        if readable:
            chunk = os.read(adapter.stdout.fileno(), 4096)
            if not chunk:
                return b""
            buffer.extend(chunk)


def turn_evidence(assistant_message, stderr_tail):
    """What the target did on a turn whose trajectory check failed. A target
    error that stops it before its instrumented boundary surfaces here."""
    try:
        stderr = stderr_tail.text().strip()
    except Exception:
        stderr = ""
    # The response and stderr share one diagnostic budget.
    response = bounded_diagnostic(redact(assistant_message, stderr_tail.redactions), DIAGNOSTIC_LIMIT // 2)
    evidence = f"adapter response for this turn: {response!r}"
    if stderr:
        evidence += f"\nadapter stderr tail: {bounded_diagnostic(stderr, DIAGNOSTIC_LIMIT // 2)}"
    return evidence


def adapter_failure(adapter, stderr_tail, message):
    """Include bounded stderr while preserving an already-written partial ATIF."""
    try:
        code = adapter.poll()
        stderr = stderr_tail.text().strip()
    except Exception:
        code, stderr = None, ""
    detail = f" ({code})" if code is not None else ""
    if stderr:
        detail += f": {stderr}"
    return RuntimeError(f"agent command failed{detail}: {message}")


def terminate_adapter_group(adapter, force=False):
    """Signal the trial-owned session, including forked adapter helpers.

    macOS reports EPERM while every remaining group member is an unreaped
    zombie. That clears once the leader is reaped here and orphaned helpers
    are reaped by init, so retry briefly; a persistent EPERM is re-raised.
    """
    import time
    deadline = time.monotonic() + 2
    while True:
        try:
            os.killpg(adapter.pid, signal.SIGKILL if force else signal.SIGTERM)
            return
        except ProcessLookupError:
            return
        except PermissionError:
            if time.monotonic() >= deadline:
                raise
            adapter.poll()
            time.sleep(0.01)


def trailing_stdout(adapter, redactions):
    """Collect only immediately available bytes; never block shutdown on a pipe."""
    import select
    output = bytearray(getattr(adapter, "_relai_stdout_buffer", b""))
    adapter._relai_stdout_buffer = bytearray()
    while True:
        readable, _, _ = select.select([adapter.stdout], [], [], 0)
        if not readable:
            break
        chunk = os.read(adapter.stdout.fileno(), 4096)
        if not chunk:
            break
        output.extend(chunk)
    return redact(output.decode(errors="replace"), redactions)


def shutdown_adapter(adapter, stderr_tail, redactions):
    """End one trial-owned adapter and reject stdout contamination or orphaning."""
    if adapter.stdin and not adapter.stdin.closed:
        adapter.stdin.close()
    try:
        code = adapter.wait(timeout=10)
    except subprocess.TimeoutExpired:
        terminate_adapter_group(adapter)
        try:
            adapter.wait(timeout=5)
        except subprocess.TimeoutExpired:
            terminate_adapter_group(adapter, force=True)
            adapter.wait(timeout=5)
        raise RuntimeError("agent command did not exit after the trial request stream closed")
    finally:
        # Even after the adapter exits normally, a helper may retain inherited
        # descriptors. End the whole trial session before inspecting pipes.
        terminate_adapter_group(adapter)
        terminate_adapter_group(adapter, force=True)
        stderr_tail.close()
    extra_stdout = trailing_stdout(adapter, redactions) if adapter.stdout else ""
    stderr = stderr_tail.text()
    if extra_stdout.strip():
        raise RuntimeError(
            "adapter stdout emitted data outside the one-response-per-request NDJSON stream: "
            f"{bounded_diagnostic(extra_stdout.strip())!r}"
        )
    if code:
        raise RuntimeError(f"agent command failed ({code}): {stderr.strip()}")


def validate_scenario(scenario):
    """Pure input validation shared with container preflight; never calls a model."""
    if not isinstance(scenario, dict):
        raise ValueError("scenario must be an object")
    input_spec = scenario.get("input")
    if not isinstance(input_spec, dict):
        raise ValueError("scenario.input must be an object")
    input_type = input_spec.get("type")
    persona_spec = None
    if input_type == "fixed":
        turns = input_spec.get("turns")
        if not isinstance(turns, list) or not all(isinstance(item, str) for item in turns):
            raise ValueError("fixed input requires scenario.input.turns strings")
    elif input_type == "persona_script":
        persona_spec = input_spec
        if not isinstance(persona_spec.get("persona"), str) or not isinstance(persona_spec.get("max_turns"), int):
            raise ValueError("persona input requires persona and positive max_turns")
        if persona_spec["max_turns"] <= 0:
            raise ValueError("persona input max_turns must be positive")
        turns = range(persona_spec["max_turns"])
    else:
        raise ValueError("scenario.input.type must be fixed or persona_script")
    services = scenario.get("services", {})
    if not isinstance(services, dict):
        raise ValueError("scenario.services must be an object")
    return turns, persona_spec, services


def load_descriptor(path):
    if path is None:
        return {}
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError("Harbor task descriptor must be an object")
    return value


def container_asset(path, label):
    if not isinstance(path, str) or not path.startswith("environment/"):
        raise ValueError(f"{label} must name an environment/ task asset")
    return Path(os.environ.get("RELAI_HARBOR_ASSET_ROOT", "/app")) / path.removeprefix("environment/")


def asset_json(path, label):
    try:
        return json.loads(container_asset(path, label).read_text(encoding="utf-8"))
    except json.JSONDecodeError as error:
        raise ValueError(f"{label} must contain JSON: {error}") from error


def task_adapter_command(descriptor, default_command):
    """Select the task-owned component/mock adapter when one is declared.

    The asset is a JSON command descriptor with the same `command` string
    semantics as `.relai/harbor/agent.toml`. Its command is intentionally
    executed by the task image: that adapter owns importing the selected
    project component and applying its declared mocks before it writes ATIF.
    """
    path = descriptor.get("component_adapter_asset")
    if path is None:
        if descriptor.get("mocks"):
            raise ValueError("mocked Harbor tasks require component_adapter_asset")
        return shlex.split(default_command), None
    adapter = asset_json(path, "component_adapter_asset")
    if not isinstance(adapter, dict) or set(adapter) != {"command"}:
        raise ValueError("component_adapter_asset must be a JSON object with only command")
    command = adapter.get("command")
    if not isinstance(command, str) or not command.strip():
        raise ValueError("component_adapter_asset.command must be a non-empty command")
    resolved = container_asset(path, "component_adapter_asset")
    # Run relative adapter commands from the verified asset directory. The
    # adapter cannot be resolved from an unchecked host or project path.
    command = shlex.split(command)
    if not command:
        raise ValueError("component_adapter_asset.command must be a non-empty command")
    return command, resolved.parent


def apply_descriptor(scenario, descriptor):
    """Resolve neutral task metadata into the leaf-adapter request contract.

    Descriptor paths identify assets copied into the task image by its own
    Dockerfile. The launcher never imports a project component or fabricates
    ATIF events; it merely passes the selected boundary and task data through.
    """
    if not descriptor:
        return scenario, descriptor
    input_type = descriptor.get("input_type")
    input_asset = descriptor.get("input_asset")
    if input_type not in {"fixed", "persona", "component_fixed", "component_generated"}:
        raise ValueError("Harbor task descriptor has unsupported input_type")
    resolved = asset_json(input_asset, "input_asset")
    if not isinstance(resolved, dict):
        raise ValueError("input_asset JSON must be an object")
    scenario = dict(scenario)
    scenario.update({key: value for key, value in resolved.items() if key != "input"})
    if input_type == "persona":
        persona = asset_json(descriptor.get("persona_asset"), "persona_asset")
        if not isinstance(persona, dict):
            raise ValueError("persona_asset JSON must be an object")
        scenario["input"] = {"type": "persona_script", **persona}
    elif input_type in {"fixed", "component_fixed", "component_generated"}:
        input_spec = resolved.get("input", resolved)
        if not isinstance(input_spec, dict):
            raise ValueError("input_asset must contain an input object")
        # Component inputs still run through the normal leaf adapter exactly
        # once per supplied turn. Its real boundary is identified below.
        scenario["input"] = {"type": "fixed", **input_spec}
    mocks = {}
    for entry in descriptor.get("mocks", []):
        if not isinstance(entry, dict) or not isinstance(entry.get("target"), str):
            raise ValueError("Harbor mock descriptor entries require target and path")
        mocks[entry["target"]] = asset_json(entry.get("path"), "mock asset")
    descriptor = dict(descriptor)
    descriptor["mocks"] = mocks
    if input_type == "component_generated":
        generated = descriptor.get("generated_component_input")
        if not isinstance(generated, dict) or not isinstance(generated.get("args"), list) or not isinstance(generated.get("kwargs"), dict):
            raise ValueError("component_generated input must be generated by the RELAI CLI before the trial")
    return scenario, descriptor


def generate_persona_turn(spec, history):
    """Ask the CLI's host relay for the persona's next turn."""
    url = os.environ.get(MODEL_RELAY_URL)
    token = os.environ.get(MODEL_RELAY_TOKEN)
    if not url or not token:
        raise RuntimeError("persona turns are generated by the RELAI CLI; run this task with relai simulate")
    payload = {
        "persona": spec["persona"],
        "history": history,
        "remaining_turns": spec["max_turns"] - len([item for item in history if item.get("role") == "user"]),
    }
    request = urllib.request.Request(
        url,
        data=json.dumps(payload).encode("utf-8"),
        headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(request) as response:
            value = json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as error:
        detail = bounded_diagnostic(error.read().decode("utf-8", "replace"))
        raise RuntimeError(f"RELAI persona relay failed with HTTP {error.code}: {detail}") from None
    except (urllib.error.URLError, OSError) as error:
        raise RuntimeError(
            f"could not reach the RELAI persona relay at {url}: {error}. "
            "Task containers must be able to reach the host as host.docker.internal."
        ) from None
    if not isinstance(value, dict) or not isinstance(value.get("should_stop"), bool):
        raise ValueError("persona relay response requires content and should_stop")
    return value


def trajectory_steps(path):
    if not path.is_file():
        raise RuntimeError("project adapter did not write RELAI_ATIF_TRAJECTORY_PATH")
    try:
        document = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as error:
        raise RuntimeError(f"project adapter wrote malformed ATIF JSON: {error}") from error
    if not isinstance(document, dict) or document.get("schema_version") != "ATIF-v1.7":
        raise RuntimeError("project adapter must write an ATIF-v1.7 trajectory")
    steps = document.get("steps")
    if not isinstance(steps, list) or not steps:
        raise RuntimeError("project adapter wrote no ATIF steps")
    return steps


if __name__ == "__main__":
    main()
