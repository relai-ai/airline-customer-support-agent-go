package airline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// relaiRunner speaks RELAI's provider-neutral protocol, not OpenAI Responses.
// The backend selects the pinned session profile; no shared provider key is used.
type relaiRunner struct {
	base, token string
	client      *http.Client
}

func newRelaiRunner() (*relaiRunner, error) {
	base, token := os.Getenv("RELAI_API_BASE"), os.Getenv("RELAI_TEAM_API_KEY")
	u, err := url.Parse(base)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !strings.HasSuffix(base, "/") || token == "" {
		return nil, fmt.Errorf("invalid RELAI session configuration")
	}
	local := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "host.docker.internal"
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return nil, fmt.Errorf("RELAI gateway requires HTTPS")
	}
	return &relaiRunner{base: base, token: token, client: &http.Client{Timeout: 120 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

type relaiCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func (r *relaiRunner) run(ctx context.Context, instructions string, tools []Tool, history []Message, out io.Writer) (string, error) {
	messages := []map[string]any{{"role": "system", "content": instructions}}
	for _, m := range history {
		if m.Role == "user" || m.Role == "assistant" {
			messages = append(messages, map[string]any{"role": m.Role, "content": m.Content})
		}
	}
	defs := []map[string]any{}
	index := map[string]Tool{}
	for _, t := range tools {
		index[t.Name] = t
		defs = append(defs, map[string]any{"name": t.Name, "description": t.Description, "input_schema": t.Parameters})
	}
	for turn := 0; turn < 16; turn++ {
		body, err := json.Marshal(map[string]any{"messages": messages, "tools": defs})
		if err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, "POST", r.base+"model-proxy/responses/", bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Token "+r.token)
		req.Header.Set("Content-Type", "application/json")
		res, err := r.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("RELAI model request unavailable")
		}
		var result struct {
			Content string      `json:"assistant_content"`
			Calls   []relaiCall `json:"tool_calls"`
		}
		if res.StatusCode != 200 {
			res.Body.Close()
			return "", fmt.Errorf("RELAI model request returned HTTP %d", res.StatusCode)
		}
		err = json.NewDecoder(io.LimitReader(res.Body, 2*1024*1024)).Decode(&result)
		res.Body.Close()
		if err != nil {
			return "", fmt.Errorf("invalid RELAI model response")
		}
		trace := recorder(ctx)
		step := -1
		if trace != nil {
			calls := []map[string]any{}
			for _, c := range result.Calls {
				calls = append(calls, map[string]any{"tool_call_id": c.ID, "function_name": c.Name, "arguments": c.Arguments})
			}
			step = trace.model(result.Content, "", calls)
		}
		if len(result.Calls) == 0 {
			if result.Content == "" {
				return "", fmt.Errorf("empty RELAI model response")
			}
			_, err = io.WriteString(out, result.Content)
			return result.Content, err
		}
		messages = append(messages, map[string]any{"role": "assistant", "content": result.Content, "tool_calls": result.Calls})
		for _, call := range result.Calls {
			tool, ok := index[call.Name]
			if !ok || call.ID == "" {
				return "", fmt.Errorf("invalid RELAI tool call")
			}
			value := tool.Execute(call.Arguments)
			trace.result(step, call.ID, value)
			messages = append(messages, map[string]any{"role": "tool", "result": map[string]any{"call_id": call.ID, "status": "success", "content": value}})
		}
	}
	return "", fmt.Errorf("RELAI tool-turn limit reached")
}
