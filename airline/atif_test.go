package airline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestATIFOrderedRedactedSnapshots(t *testing.T) {
	t.Setenv("RELAI_ATIF_TRAJECTORY_PATH", filepath.Join(t.TempDir(), "trajectory.json"))
	t.Setenv("OPENAI_API_KEY", "test-sensitive-value")
	r := newATIFRecorder()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); r.user("hello test-sensitive-value") }()
	}
	wg.Wait()
	step := r.model("", "observed-test-model", []map[string]any{{"tool_call_id": "test-call", "function_name": "test-fixture", "arguments": map[string]any{"AuThOrIzAtIoN": "private", "safe": "value"}}})
	r.result(step, "test-call", `{"password":"hidden"}`)
	b, err := os.ReadFile(r.path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "test-sensitive-value") || strings.Contains(string(b), "hidden") || strings.Contains(string(b), "private") {
		t.Fatal("sensitive value leaked")
	}
	var doc struct {
		Steps []struct {
			ID int `json:"step_id"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Steps) != 11 {
		t.Fatal("lost snapshot prefix")
	}
	for i, s := range doc.Steps {
		if s.ID != i+1 {
			t.Fatal("unordered steps")
		}
	}
}
