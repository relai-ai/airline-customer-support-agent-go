package airline

// Optional RELAI tracing. All mutations and synchronous snapshots share one lock;
// no producer can overtake another snapshot, and no writes remain to drain.
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type atifKey struct{}
type atifRecorder struct {
	mu      sync.Mutex
	path    string
	steps   []map[string]any
	secrets []string
}

func newATIFRecorder() *atifRecorder {
	path := os.Getenv("RELAI_ATIF_TRAJECTORY_PATH")
	if path == "" {
		return nil
	}
	r := &atifRecorder{path: path}
	// Exact sensitive-name allowlist matching .relai/runtime-manifest.json.
	for _, name := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY"} {
		if value := os.Getenv(name); value != "" {
			r.secrets = append(r.secrets, value)
		}
	}
	return r
}
func recorder(ctx context.Context) *atifRecorder {
	r, _ := ctx.Value(atifKey{}).(*atifRecorder)
	return r
}

var sensitiveKey = regexp.MustCompile(`(?i)(authorization|cookie|token|password|passwd|secret|api[_-]?key|credential|private[_-]?key)`)
var sensitiveText = regexp.MustCompile(`(?i)(["']?(?:[a-z0-9_-]*(?:api[_-]?key|token|password|passwd|secret|credential)|authorization|cookie)["']?\s*[:=]\s*)(?:"[^"\n]*"|'[^'\n]*'|[^\s,;}]+(?:[ \t]+[^\r\n,;}]+)?)`)
var privateKey = regexp.MustCompile(`(?s)-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----`)
var bearer = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/-]+=*`)
var keyBlob = regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{12,}`)

func (r *atifRecorder) filter(v any) any {
	switch x := v.(type) {
	case map[string]any:
		y := make(map[string]any, len(x))
		for k, v := range x {
			if sensitiveKey.MatchString(k) {
				y[k] = "[REDACTED]"
			} else {
				y[k] = r.filter(v)
			}
		}
		return y
	case []any:
		y := make([]any, len(x))
		for i, v := range x {
			y[i] = r.filter(v)
		}
		return y
	case string:
		for _, secret := range r.secrets {
			x = strings.ReplaceAll(x, secret, "[REDACTED]")
		}
		x = privateKey.ReplaceAllString(x, "[REDACTED PRIVATE KEY]")
		x = sensitiveText.ReplaceAllString(x, "${1}[REDACTED]")
		x = bearer.ReplaceAllString(x, "Bearer [REDACTED]")
		return keyBlob.ReplaceAllString(x, "[REDACTED]")
	default:
		return v
	}
}
func (r *atifRecorder) snapshot() {
	// Normalize every field recursively at the final serialization boundary.
	doc := map[string]any{"schema_version": "ATIF-v1.7", "agent": map[string]any{"name": AgentName, "version": "0.1.0"}, "steps": r.steps}
	raw, err := json.Marshal(doc)
	var normalized any
	if err == nil {
		err = json.Unmarshal(raw, &normalized)
	}
	if err == nil {
		raw, err = json.Marshal(r.filter(normalized))
	}
	if err == nil {
		err = os.MkdirAll(filepath.Dir(r.path), 0755)
	}
	if err == nil {
		var f *os.File
		f, err = os.CreateTemp(filepath.Dir(r.path), ".atif-*.json")
		if err == nil {
			name := f.Name()
			defer os.Remove(name)
			_, err = f.Write(raw)
			if err == nil {
				err = f.Sync()
			}
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
			if err == nil {
				err = os.Rename(name, r.path)
			}
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "ATIF recording failed:", r.filter(err.Error()))
	}
}
func (r *atifRecorder) user(message string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steps = append(r.steps, map[string]any{"step_id": len(r.steps) + 1, "source": "user", "message": message})
	r.snapshot()
}

// model records an observed provider message before any of its calls dispatch.
func (r *atifRecorder) model(text, model string, calls []map[string]any) int {
	if r == nil {
		return -1
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if text == "" && len(calls) > 0 {
		b, err := json.Marshal(calls)
		if err != nil {
			return -1
		}
		text = string(b)
	}
	if text == "" {
		return -1
	}
	step := map[string]any{"step_id": len(r.steps) + 1, "source": "agent", "message": text, "model_name": model}
	if len(calls) > 0 {
		step["tool_calls"] = calls
	}
	r.steps = append(r.steps, step)
	r.snapshot()
	return len(r.steps) - 1
}
func (r *atifRecorder) result(step int, id, output string) {
	if r == nil || step < 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.steps[step]
	obs, _ := s["observation"].(map[string]any)
	if obs == nil {
		obs = map[string]any{"results": []any{}}
		s["observation"] = obs
	}
	obs["results"] = append(obs["results"].([]any), map[string]any{"source_call_id": id, "content": output})
	r.snapshot()
}
func observedArguments(raw string) any {
	var args any
	if json.Unmarshal([]byte(raw), &args) == nil {
		return args
	}
	return map[string]any{"raw": raw}
}
