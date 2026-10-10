package airline

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRelaiGatewayToolRoundtrip(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/runtime/test/model-proxy/responses/" || r.Header.Get("Authorization") != "Token pg_test" {
			t.Errorf("unexpected scoped request")
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["model"]; ok {
			t.Error("must not override backend profile")
		}
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			io.WriteString(w, `{"assistant_content":"","tool_calls":[{"id":"call-1","name":"lookup","arguments":{"code":"SKY123"}}]}`)
		} else {
			messages := body["messages"].([]any)
			last := messages[len(messages)-1].(map[string]any)
			if last["role"] != "tool" {
				t.Error("missing tool result")
			}
			io.WriteString(w, `{"assistant_content":"Your seat is 12A.","tool_calls":[]}`)
		}
	}))
	defer server.Close()
	t.Setenv("RELAI_API_BASE", server.URL+"/runtime/test/")
	t.Setenv("RELAI_TEAM_API_KEY", "pg_test")
	t.Setenv("AIRLINE_MODEL_PROVIDER", "relai")
	runner, err := newRelaiRunner()
	if err != nil {
		t.Fatal(err)
	}
	result, err := runner.run(context.Background(), "Help with flights", []Tool{{Name: "lookup", Parameters: map[string]any{"type": "object"}, Execute: func(map[string]any) string { return "12A" }}}, []Message{{Role: "user", Content: "My seat?"}}, io.Discard)
	if err != nil || result != "Your seat is 12A." || calls != 2 {
		t.Fatalf("result=%q calls=%d error=%v", result, calls, err)
	}
}
func TestRelaiGatewayRejectsRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "https://example.com", 302) }))
	defer server.Close()
	t.Setenv("RELAI_API_BASE", server.URL+"/")
	t.Setenv("RELAI_TEAM_API_KEY", "pg_test")
	runner, err := newRelaiRunner()
	if err != nil {
		t.Fatal(err)
	}
	_, err = runner.run(context.Background(), "help", nil, []Message{{Role: "user", Content: "hi"}}, io.Discard)
	if err == nil {
		t.Fatal("redirect accepted")
	}
}
