"""Build the mounted Go target, then become the persistent NDJSON adapter."""
import os
from pathlib import Path
import subprocess
import tempfile

os.chdir('/workspace')
env = os.environ.copy()
env.update(GOCACHE='/tmp/relai-go-build', GOMODCACHE='/go/pkg/mod', GOPROXY='off')
# Unique output outside the read-only project mount. Never reuse old binaries.
output = Path(tempfile.mkdtemp(prefix='airline-adapter-')) / 'adapter'
subprocess.run(['go', 'build', '-mod=readonly', '-o', str(output), '/app/project_adapter.go'], env=env, check=True, stdout=__import__('sys').stderr)
os.execve(str(output), [str(output)], env)
