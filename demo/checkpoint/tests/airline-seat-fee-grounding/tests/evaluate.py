#!/usr/bin/env python3
"""Minimal deterministic verifier for a Harbor adapter ATIF trajectory."""
import json
from pathlib import Path

trajectory = json.loads(Path("/logs/agent/trajectory.json").read_text(encoding="utf-8"))
assert trajectory["schema_version"] == "ATIF-v1.7"
assert trajectory["agent"]["name"]
assert trajectory["agent"]["version"]
assert trajectory["steps"]
assert [step["step_id"] for step in trajectory["steps"]] == list(range(1, len(trajectory["steps"]) + 1))
assert any(step["source"] == "agent" for step in trajectory["steps"])
