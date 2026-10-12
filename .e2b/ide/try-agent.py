"""Run the real sample adapter with a traveler request passed as JSON data."""
import base64
import json
import subprocess
import sys

try:
    request = json.loads(base64.b64decode(sys.argv[1], validate=True))
    result = subprocess.run(
        ["go", "run", "./.relai/harbor/adapter/project_adapter.go"],
        cwd="/home/user/airline",
        input=json.dumps(request) + "\n",
        text=True,
        capture_output=True,
        timeout=150,
    )
    if result.returncode:
        raise RuntimeError("Adapter failed")
    reply = json.loads(result.stdout.strip())
    if not isinstance(reply.get("assistant_message"), str):
        raise RuntimeError("No reply")
    print(json.dumps({"assistant_message": reply["assistant_message"]}))
except Exception:
    # Provider errors can include credentials. Never echo error bodies.
    print("The airline agent could not respond. Check model access and retry.", file=sys.stderr)
    sys.exit(1)
