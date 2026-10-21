package airline

// Optional recording at the production model/dispatch boundary. All writes are
// synchronous under one mutex, so no callback can publish an older snapshot.
import (
 "context"
 "crypto/rand"
 "encoding/hex"
 "encoding/json"
 "fmt"
 "os"
 "path/filepath"
 "regexp"
 "strings"
 "sync"
)

type traceKey struct{}
type relaiTrace struct {
 mu sync.Mutex
 path string
 doc map[string]any
 steps []any
 secrets []string
}

func newRelaiTrace() *relaiTrace {
 path := os.Getenv("RELAI_ATIF_TRAJECTORY_PATH")
 if path == "" { return nil }
 id := make([]byte, 16)
 if _, err := rand.Read(id); err != nil { fmt.Fprintln(os.Stderr, "ATIF: cannot generate session ID"); return nil }
 t := &relaiTrace{path:path, doc:map[string]any{"schema_version":"ATIF-v1.7", "session_id":hex.EncodeToString(id), "agent":map[string]any{"name":AgentName,"version":"0.1.0"}}}
 // Exact sensitive-name allowlist from .relai/runtime-manifest.json. Never
 // enumerate the environment to discover secrets.
 for _, name := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY"} {
  if value := os.Getenv(name); value != "" { t.secrets = append(t.secrets, value) }
 }
 return t
}

var sensitiveKey = regexp.MustCompile(`(?i)(authorization|cookie|token|password|passwd|secret|api[_-]?key|credential|private[_-]?key)`)
var secretText = regexp.MustCompile(`(?i)(["']?(?:authorization|cookie|[a-z0-9_]*(?:token|password|secret|api[_-]?key|credential))["']?\s*[:=]\s*)(?:"[^"]*"|'[^']*'|[^\s,;}]+)`)
var bearerText = regexp.MustCompile(`(?i)\b(bearer|basic)\s+[A-Za-z0-9+/_.=-]+`)
var privateKeyText = regexp.MustCompile(`(?s)-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----`)
func (t *relaiTrace) filter(v any) any {
 switch x := v.(type) {
 case map[string]any:
  out := map[string]any{}
  for k, value := range x { if sensitiveKey.MatchString(k) { out[k] = "[REDACTED]" } else { out[k] = t.filter(value) } }
  return out
 case []any:
  out := make([]any,len(x)); for i, value := range x { out[i] = t.filter(value) }; return out
 case string:
  for _, value := range t.secrets { x = strings.ReplaceAll(x,value,"[REDACTED]") }
  // Parse serialized objects too, keeping their useful nonsensitive structure.
  var decoded any
  if json.Unmarshal([]byte(x), &decoded) == nil {
   switch decoded.(type) { case map[string]any, []any: b, err := json.Marshal(t.filter(decoded)); if err == nil { return string(b) } }
  }
  x = privateKeyText.ReplaceAllString(x,"[REDACTED PRIVATE KEY]")
  x = bearerText.ReplaceAllString(x,"$1 [REDACTED]")
  return secretText.ReplaceAllString(x,"${1}[REDACTED]")
 default: return v
 }
}

// Caller holds mu for the entire append/read/serialize/replace transaction.
func (t *relaiTrace) save() {
 t.doc["steps"] = t.steps
 data, err := json.Marshal(t.filter(t.doc)) // final serialization boundary
 if err == nil { err = os.MkdirAll(filepath.Dir(t.path),0700) }
 if err == nil {
  var f *os.File
  f, err = os.CreateTemp(filepath.Dir(t.path), ".atif-*")
  if err == nil {
   name := f.Name()
   defer os.Remove(name)
   _, err = f.Write(data)
   if err == nil { err = f.Sync() }
   closeErr := f.Close(); if err == nil { err = closeErr }
   if err == nil { err = os.Rename(name,t.path) }
  }
 }
 if err != nil { fmt.Fprintln(os.Stderr,"ATIF snapshot write failed:", t.filter(err.Error())) }
}
func (t *relaiTrace) step(source, message, model string) int {
 if t == nil { return -1 }
 t.mu.Lock(); defer t.mu.Unlock()
 s := map[string]any{"step_id":len(t.steps)+1,"source":source,"message":message}
 if model != "" { s["model_name"] = model }
 t.steps = append(t.steps,t.filter(s)); t.save(); return len(t.steps)-1
}
func traceFrom(ctx context.Context) *relaiTrace { t, _ := ctx.Value(traceKey{}).(*relaiTrace); return t }
func traceModel(ctx context.Context, text string, observed any, model string) int {
 // Tool-only messages may have no text. Serialize their observed structured
 // SDK message (stream accumulators need not retain RawJSON).
 if text == "" {
  data, err := json.Marshal(observed)
  if err != nil { fmt.Fprintln(os.Stderr, "ATIF model serialization failed"); return -1 }
  text = string(data)
 }
 return traceFrom(ctx).step("agent",text,model)
}
func traceCall(ctx context.Context, step int, id, name string, args any) {
 t := traceFrom(ctx); if t == nil || step < 0 { return }
 t.mu.Lock(); defer t.mu.Unlock()
 s := t.steps[step].(map[string]any)
 calls, _ := s["tool_calls"].([]any)
 s["tool_calls"] = append(calls,t.filter(map[string]any{"tool_call_id":id,"function_name":name,"arguments":args}))
 t.save()
}
func traceResult(ctx context.Context, step int, id, result string) {
 t := traceFrom(ctx); if t == nil || step < 0 { return }
 t.mu.Lock(); defer t.mu.Unlock()
 s := t.steps[step].(map[string]any)
 obs, _ := s["observation"].(map[string]any); if obs == nil { obs = map[string]any{}; s["observation"] = obs }
 results, _ := obs["results"].([]any)
 obs["results"] = append(results,t.filter(map[string]any{"source_call_id":id,"content":result}))
 t.save()
}
