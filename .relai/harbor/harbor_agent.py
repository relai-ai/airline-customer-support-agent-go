"""Harbor bridge for a RELAI project adapter."""
import base64
import inspect
import json
import os
import shlex
import tempfile
from pathlib import Path
from typing import override

from harbor.agents.base import BaseAgent
from harbor.environments.base import BaseEnvironment
from harbor.models.agent.context import AgentContext
from harbor.utils.trajectory_validator import TrajectoryValidator


_TRAJECTORY_PATH = "/logs/agent/trajectory.json"
_AGENT_LOGS = "/logs/agent"
# Protects the host from unbounded container output copied through the bridge.
_MAX_FALLBACK_ARTIFACT_BYTES = 16 * 1024 * 1024
# Head-and-tail diagnostic size; must match DIAGNOSTIC_BYTES in src/harbor/mod.rs.
_DIAGNOSTIC_LIMIT = 16 * 1024
# Set by the CLI on the Harbor process when a persona trial needs model turns.
_MODEL_RELAY_ENV = ("RELAI_MODEL_RELAY_URL", "RELAI_MODEL_RELAY_TOKEN")
_LAUNCHER_CONTROL_ENV = ("RELAI_HARBOR_USER_INPUT_STEP_MODE",)
_DOCKER_READONLY_COPY_SIGNATURE = ("openat workspace/", "read-only file system")
_PREFLIGHT = r'''import json, os, pathlib, runpy, shlex, shutil, subprocess, tempfile

config = json.loads(os.environ.get("RELAI_HARBOR_PREFLIGHT", "{}"))
runtime_only = bool(config.get("runtime_only"))
descriptor = json.loads(os.environ.get("RELAI_HARBOR_TASK_DESCRIPTOR", "{}"))
deferred = (not runtime_only and bool(descriptor.get("benchmark_sample"))
            and os.environ.get("RELAI_HARBOR_PREFLIGHT_PHASE") != "post_materialization")

for name in ["python3", *([] if deferred else config.get("executables", []))]:
    if not isinstance(name, str) or not shutil.which(name):
        raise RuntimeError("missing required runtime executable: " + str(name))

if not runtime_only:
    for value in ["/workspace", "/app/relai-harbor/adapter/run.py", "/app/relai-harbor/scenario.json"]:
        if not pathlib.Path(value).exists():
            raise RuntimeError("missing runtime path: " + value)

for value in ([] if deferred else config.get("assets", [])):
    if not isinstance(value, str) or not pathlib.Path(value).is_file():
        label = "runtime asset" if runtime_only else "packaging asset"
        raise RuntimeError("missing configured " + label + ": " + str(value))

if not runtime_only:
    # Benchmark assets are written by its materializer, checked by the launcher
    # after materialization; do not reject them before they can exist.
    if not deferred:
        assets = [descriptor.get(key) for key in ("input_asset", "persona_asset", "component_adapter_asset")]
        assets.extend(item.get("path") for item in descriptor.get("mocks", []))
        for value in filter(None, assets):
            relative = pathlib.PurePosixPath(value)
            if relative.is_absolute() or ".." in relative.parts or not relative.parts or relative.parts[0] != "environment":
                raise RuntimeError("invalid task asset path")
            path = pathlib.Path("/app").joinpath(*relative.parts[1:])
            if not path.is_file():
                raise RuntimeError("missing task asset: " + str(path))
    for directory in ("/logs/agent", "/tmp"):
        pathlib.Path(directory).mkdir(parents=True, exist_ok=True)
        with tempfile.TemporaryFile(dir=directory) as output:
            output.write(b"preflight")

if not deferred:
    if runtime_only:
        argv = shlex.split(os.environ.get("RELAI_HARBOR_AGENT_COMMAND", ""))
        cwd = None
    else:
        launcher = runpy.run_path("/app/relai-harbor/adapter/run.py")
        scenario = json.loads(pathlib.Path("/app/relai-harbor/scenario.json").read_text())
        if descriptor.get("input_type") != "component_generated":
            scenario, _ = launcher["apply_descriptor"](scenario, descriptor)
            launcher["validate_scenario"](scenario)
        argv, cwd = launcher["task_adapter_command"](
            descriptor, os.environ.get("RELAI_HARBOR_AGENT_COMMAND", ""))
    if not argv:
        raise RuntimeError("configured adapter command is empty")
    executable = argv[0]
    if cwd is not None:
        os.chdir(cwd)
    resolved = shutil.which(executable)
    if resolved is None:
        raise RuntimeError("missing adapter executable in configured container: " + executable)
    for index, command in enumerate(config.get("commands", [])):
        marker = "RELAI_SHARED_RUNTIME_PROBE_FAILED: " if config.get("shared_runtime_probes") else ""
        result = subprocess.run(command, shell=True, cwd=cwd, capture_output=True, text=True)
        if result.returncode:
            label = "shared runtime dependency probe " if runtime_only else "dependency probe "
            raise RuntimeError(marker + label + str(index) + " failed in configured container; adapter executable " + resolved + "; use the adapter runtime for import checks: " + (result.stderr or result.stdout))
'''
_PREFLIGHT_PYTHON_GUARD = r'''command -v python3 >/dev/null 2>&1 || { echo 'RELAI Harbor requires python3 for its launcher; provide it in the image or runtime setup.' >&2; exit 1; }'''
_BENCHMARK_MATERIALIZER = r'''
import json, os, subprocess
descriptor = json.loads(os.environ.get("RELAI_HARBOR_TASK_DESCRIPTOR", "{}"))
value = descriptor.get("benchmark_sample")
if value is not None:
    command = value.get("materializer")
    row = value.get("row")
    if not isinstance(command, str) or not command.strip() or not isinstance(row, dict):
        raise ValueError("invalid Harbor benchmark sample asset")
    completed = subprocess.run(command, shell=True, input=json.dumps(row), text=True,
                               capture_output=True, check=False)
    if completed.returncode:
        raise RuntimeError("Harbor benchmark materializer failed: " + completed.stderr)
'''

_STAGE_DESCRIPTOR = r'''
import os, pathlib
pathlib.Path("/tmp/relai-harbor-task.json").write_text(
    os.environ["RELAI_HARBOR_TASK_DESCRIPTOR"], encoding="utf-8")
'''


class HarborAdapterAgent(BaseAgent):
    @staticmethod
    @override
    def name() -> str:
        return "relai-harbor-adapter"

    @override
    def version(self) -> str:
        return "0.1.0"

    @override
    async def setup(self, environment: BaseEnvironment) -> None:
        """The task image owns all project adapter dependencies."""
        # Harbor calls setup once the trial environment is built and started.
        # The CLI's run-phase watchdog starts when this marker appears.
        marker = os.environ.get("RELAI_HARBOR_RUN_PHASE_MARKER")
        if marker:
            Path(marker).touch()

    @override
    async def run(self, instruction: str, environment: BaseEnvironment, context: AgentContext) -> None:
        agent_command = self.extra_env.get("RELAI_HARBOR_AGENT_COMMAND")
        if not agent_command:
            raise ValueError("RELAI_HARBOR_AGENT_COMMAND is required")
        command = " ".join((
            "python3", "/app/relai-harbor/adapter/run.py",
            "--scenario", "/app/relai-harbor/scenario.json",
            "--trajectory", _TRAJECTORY_PATH,
            "--agent-command", shlex.quote(agent_command),
        ))
        names = self.extra_env.get("RELAI_HARBOR_RUNTIME_ENV_NAMES", "").split(",")
        runtime_env = {name: os.environ[name] for name in names if name and name in os.environ}
        forwarded_names = list(runtime_env)
        descriptor = self.extra_env.get("RELAI_HARBOR_TASK_DESCRIPTOR")
        # Probe in the actual built image with the real read-only project mount.
        # Do not forward model credentials to import-only probes.
        preflight_env = {name: "" for name in runtime_env}
        # Harbor's scoped agent env overrides per-exec values. Keep the command
        # in its canonical dedicated variable rather than enriching scoped JSON.
        preflight_env["RELAI_HARBOR_AGENT_COMMAND"] = agent_command
        preflight_env["RELAI_HARBOR_PREFLIGHT"] = self.extra_env.get("RELAI_HARBOR_PREFLIGHT", "{}")
        if descriptor:
            preflight_env["RELAI_HARBOR_TASK_DESCRIPTOR"] = descriptor
        # This must stay a shell command: cached and custom images can lack
        # Python, so launching the Python preflight itself would hide the
        # actionable requirement behind a command-not-found failure.
        try:
            python_check = await environment.exec(
                command="sh -c " + shlex.quote(
                    _PREFLIGHT_PYTHON_GUARD
                ), env=preflight_env
            )
        except Exception as error:
            detail = bounded_diagnostic(redact_runtime_values(str(error), runtime_env))
            raise RuntimeError("RELAI_PREFLIGHT_FAILED: " + detail) from None
        if python_check.return_code != 0:
            raise RuntimeError("RELAI_PREFLIGHT_FAILED: " + adapter_diagnostic(python_check, runtime_env))
        try:
            preflight = await environment.exec(
                command="python3 -c " + shlex.quote(_PREFLIGHT), env=preflight_env
            )
        except Exception as error:
            detail = bounded_diagnostic(redact_runtime_values(str(error), runtime_env))
            raise RuntimeError("RELAI_PREFLIGHT_FAILED: " + detail) from None
        if preflight.return_code != 0:
            raise RuntimeError("RELAI_PREFLIGHT_FAILED: " + adapter_diagnostic(preflight, runtime_env))
        if descriptor:
            runtime_env["RELAI_HARBOR_TASK_DESCRIPTOR"] = descriptor
            staged = await environment.exec(
                command="python3 -c " + shlex.quote(_STAGE_DESCRIPTOR), env=runtime_env
            )
            if staged.return_code != 0:
                raise RuntimeError("could not stage Harbor task descriptor: " + adapter_diagnostic(staged, runtime_env))
            command += " --descriptor /tmp/relai-harbor-task.json"
        # A benchmark row is materialized only inside the task image. The CLI
        # writes its immutable CSV-derived asset but never executes the
        # agent-authored materializer on the host.
        materialized = await environment.exec(
            command="python3 -c " + shlex.quote(_BENCHMARK_MATERIALIZER), env=runtime_env
        )
        if materialized.return_code != 0:
            raise RuntimeError("RELAI_MATERIALIZATION_FAILED: " + adapter_diagnostic(materialized, runtime_env))
        if descriptor and json.loads(descriptor).get("benchmark_sample"):
            preflight_env["RELAI_HARBOR_PREFLIGHT_PHASE"] = "post_materialization"
            try:
                checked = await environment.exec(
                    command="python3 -c " + shlex.quote(_PREFLIGHT), env=preflight_env
                )
            except Exception as error:
                detail = bounded_diagnostic(redact_runtime_values(str(error), runtime_env))
                raise RuntimeError("RELAI_PREFLIGHT_FAILED: " + detail) from None
            if checked.return_code != 0:
                raise RuntimeError("RELAI_PREFLIGHT_FAILED: " + adapter_diagnostic(checked, runtime_env))
        # The CLI's persona relay is for the launcher alone: never the probes,
        # staging, or materializer execs above.
        launcher_env = dict(runtime_env)
        # The launcher redacts these forwarded values from its own bounded
        # diagnostics before any truncation boundary can split one.
        launcher_env["RELAI_HARBOR_RUNTIME_ENV_NAMES"] = ",".join(forwarded_names)
        for name in _MODEL_RELAY_ENV:
            if os.environ.get(name):
                launcher_env[name] = os.environ[name]
        for name in _LAUNCHER_CONTROL_ENV:
            if self.extra_env.get(name):
                launcher_env[name] = self.extra_env[name]
        result = await environment.exec(command=command, env=launcher_env)
        trajectory_path = self.logs_dir / "trajectory.json"
        if not trajectory_path.is_file():
            try:
                await environment.download_file(_TRAJECTORY_PATH, trajectory_path)
            except Exception:
                # The normal Docker path bind-mounts /logs into logs_dir. Remote
                # environments need an explicit transfer before validation.
                pass
        trajectory_error = validate_trajectory(trajectory_path, runtime_env)
        if result.return_code != 0:
            suffix = f"; secondary trajectory validation: {trajectory_error}" if trajectory_error else ""
            raise RuntimeError(
                f"RELAI_ADAPTER_FAILED: project adapter exited with status {result.return_code}: "
                f"{adapter_diagnostic(result, launcher_env)}{suffix}"
            )
        if trajectory_error:
            raise RuntimeError("RELAI_TRAJECTORY_FAILED: " + trajectory_error)
        self._install_artifact_download_fallback(environment)
        context.metadata = {"trajectory": _TRAJECTORY_PATH, "instruction_received": bool(instruction.strip())}


    def _install_artifact_download_fallback(self, environment: BaseEnvironment) -> None:
        original_download_file = environment.download_file
        artifacts_dir = self.logs_dir.parent / "artifacts"

        async def download_file(source_path: str, target_path) -> None:
            relative = _agent_log_file(source_path)
            target = Path(target_path)
            if relative is None or not _is_trial_artifact_target(target, artifacts_dir):
                await original_download_file(source_path, target_path)
                return
            # Local Docker bind-mounts /logs/agent into logs_dir, so a declared
            # agent log file is already on the host. Copying it here skips
            # Docker's cp, which cannot set up the read-only /workspace mount
            # and its nested masks. The container writes these files, so the
            # host read refuses symlinks; anything it cannot read falls back.
            try:
                mounted = read_agent_log_file(str(self.logs_dir), relative, _MAX_FALLBACK_ARTIFACT_BYTES)
            except (OSError, ValueError):
                mounted = None
            if mounted is not None:
                write_atomically(target, mounted)
                return
            try:
                await original_download_file(source_path, target_path)
                return
            except Exception as error:
                if not _is_readonly_copy_error(error):
                    raise
                fallback_error = error
            result = await environment.exec(
                command="python3 -c " + shlex.quote(_READ_ARTIFACT) + " " + shlex.quote(relative),
                user="root",
            )
            if result.return_code != 0:
                raise fallback_error
            try:
                value = base64.b64decode(result.stdout or "", validate=True)
            except Exception:
                raise fallback_error
            if len(value) > _MAX_FALLBACK_ARTIFACT_BYTES:
                raise fallback_error
            write_atomically(target, value)

        environment.download_file = download_file


def write_atomically(path: Path, value: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(dir=path.parent, delete=False) as target:
            temporary = target.name
            target.write(value)
            target.flush()
            os.chmod(temporary, 0o644)
            os.fsync(target.fileno())
        os.replace(temporary, path)
        temporary = None
    finally:
        if temporary is not None:
            os.unlink(temporary)


def _is_readonly_copy_error(error: Exception) -> bool:
    message = str(error).lower()
    return all(marker in message for marker in _DOCKER_READONLY_COPY_SIGNATURE)


def read_agent_log_file(root: str, relative: str, maximum: int) -> bytes:
    """Read a regular file below root without following any symlink.

    Each parent directory is pinned by descriptor, so a concurrent swap of a
    parent for a symlink cannot redirect the read. The container fallback runs
    this same function's source, so it must stay self-contained.
    """
    import os
    import stat

    components = relative.split("/")
    directory_fds = [os.open(root, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)]
    try:
        for component in components[:-1]:
            directory_fds.append(
                os.open(component, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=directory_fds[-1])
            )
        source_fd = os.open(components[-1], os.O_RDONLY | os.O_NOFOLLOW, dir_fd=directory_fds[-1])
        with os.fdopen(source_fd, "rb") as source:
            metadata = os.fstat(source.fileno())
            if not stat.S_ISREG(metadata.st_mode):
                raise ValueError("artifact is not a regular file")
            if metadata.st_size > maximum:
                raise ValueError("artifact exceeds transfer limit")
            value = source.read(maximum + 1)
    finally:
        for directory_fd in reversed(directory_fds):
            os.close(directory_fd)
    if len(value) > maximum:
        raise ValueError("artifact exceeds transfer limit")
    return value


_READ_ARTIFACT = (
    inspect.getsource(read_agent_log_file)
    + "\nimport base64, sys\n"
    + f"value = read_agent_log_file({_AGENT_LOGS!r}, sys.argv[1], {_MAX_FALLBACK_ARTIFACT_BYTES})\n"
    + "sys.stdout.write(base64.b64encode(value).decode('ascii'))\n"
)


def _agent_log_file(source_path: str) -> str | None:
    """Return the path below /logs/agent for a file source, or None.

    Mirrors the task artifact rule in the CLI's Harbor task validation.
    """
    prefix = _AGENT_LOGS + "/"
    if not source_path.startswith(prefix):
        return None
    relative = source_path[len(prefix):]
    if any(component in ("", ".", "..") for component in relative.split("/")):
        return None
    return relative


def _is_trial_artifact_target(target: Path, artifacts_dir: Path) -> bool:
    try:
        relative = target.relative_to(artifacts_dir)
    except ValueError:
        return False
    if not relative.parts or any(component in (".", "..") for component in relative.parts):
        return False
    current = artifacts_dir
    for component in relative.parts:
        current /= component
        if current.is_symlink():
            return False
    return True


def validate_trajectory(path, runtime_env: dict[str, str]) -> str | None:
    """Require the adapter to persist a complete upstream ATIF document.

    The project instrumentation owns every step and metric in this file.  The
    bridge only validates Harbor's canonical model; it never reconstructs a
    trajectory from launcher control output.
    """
    if not path.is_file():
        return "project adapter did not write /logs/agent/trajectory.json as a canonical ATIF trajectory"
    validator = TrajectoryValidator()
    if validator.validate(path):
        return None
    details = redact_runtime_values("; ".join(validator.get_errors()), runtime_env)
    return (
        "project adapter wrote an invalid ATIF trajectory at /logs/agent/trajectory.json: "
        f"{details}"
    )


def adapter_diagnostic(result, runtime_env: dict[str, str], limit: int = _DIAGNOSTIC_LIMIT) -> str:
    """Keep failed adapter output useful without exposing forwarded values.

    Each stream gets its own share of the budget so a noisy stdout cannot
    erase the cause retained from stderr.
    """
    streams = []
    for label in ("stderr", "stdout"):
        output = (getattr(result, label, None) or "").strip()
        if output:
            streams.append((label, redact_runtime_values(output, runtime_env)))
    if not streams:
        return "adapter returned no output"
    remaining = limit - sum(len(label) + 2 for label, _ in streams)
    chunks = []
    for index, (label, text) in enumerate(streams):
        retained = bounded_diagnostic(text, remaining // (len(streams) - index))
        remaining -= len(retained)
        chunks.append(f"{label}={retained}")
    return " ".join(chunks)


def bounded_diagnostic(text: str, limit: int = _DIAGNOSTIC_LIMIT) -> str:
    """Keep the initial cause and the final status within `limit` characters."""
    marker = "\n... [truncated] ...\n"
    if len(text) <= limit:
        return text
    kept = max(0, limit - len(marker))
    return text[:kept // 2] + marker + text[len(text) - (kept - kept // 2):]


def redact_runtime_values(diagnostic: str, runtime_env: dict[str, str]) -> str:
    """Remove only values forwarded into the agent phase from diagnostics."""
    values = {value for name, value in runtime_env.items() if name != "RELAI_HARBOR_RUNTIME_ENV_NAMES"}
    for value in sorted(values, key=len, reverse=True):
        if not value:
            continue
        variants = (value, json.dumps(value)[1:-1])
        for variant in sorted(set(variants), key=len, reverse=True):
            diagnostic = diagnostic.replace(variant, "[redacted]")
    return diagnostic
