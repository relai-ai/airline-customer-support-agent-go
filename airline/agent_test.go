package airline

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestStreamResponseAppendsSignoffToStreamAndReturn(t *testing.T) {
	agent := testAgentReturning("Your carry-on allowance depends on the fare.")
	var out bytes.Buffer

	response, err := agent.StreamResponse(context.Background(), nil, &out)
	if err != nil {
		t.Fatalf("StreamResponse returned error: %v", err)
	}

	want := "Your carry-on allowance depends on the fare.\n\n" + DefaultCustomerServiceSignoff
	if response != want {
		t.Fatalf("returned response = %q, want %q", response, want)
	}
	if out.String() != want {
		t.Fatalf("streamed response = %q, want %q", out.String(), want)
	}
}

func TestStreamResponseDoesNotDuplicateEquivalentSignoff(t *testing.T) {
	base := "Your ticket includes one personal item.\n\nPLEASE, let me know if you have any questions!"
	agent := testAgentReturning(base)
	var out bytes.Buffer

	response, err := agent.StreamResponse(context.Background(), nil, &out)
	if err != nil {
		t.Fatalf("StreamResponse returned error: %v", err)
	}
	if response != base {
		t.Fatalf("returned response = %q, want %q", response, base)
	}
	if out.String() != base {
		t.Fatalf("streamed response = %q, want %q", out.String(), base)
	}
}

func TestStreamResponseLeavesEmptyOutputUnchanged(t *testing.T) {
	agent := testAgentReturning("")
	var out bytes.Buffer

	response, err := agent.StreamResponse(context.Background(), nil, &out)
	if err != nil {
		t.Fatalf("StreamResponse returned error: %v", err)
	}
	if response != "" || out.String() != "" {
		t.Fatalf("empty response changed: returned %q, streamed %q", response, out.String())
	}
}

func TestStreamResponseDoesNotFinishRunnerError(t *testing.T) {
	runnerErr := errors.New("runner failed")
	agent := NewAgentWithRunnerFunc(func(context.Context, string, []Tool, []Message, io.Writer) (string, error) {
		return "partial response", runnerErr
	})
	var out bytes.Buffer

	response, err := agent.StreamResponse(context.Background(), nil, &out)
	if !errors.Is(err, runnerErr) {
		t.Fatalf("StreamResponse error = %v, want %v", err, runnerErr)
	}
	if response != "partial response" {
		t.Fatalf("returned response = %q, want runner response", response)
	}
	if strings.Contains(out.String(), DefaultCustomerServiceSignoff) {
		t.Fatalf("streamed response unexpectedly contains signoff: %q", out.String())
	}
}

func TestStreamResponseReturnsSignoffWriterError(t *testing.T) {
	writerErr := errors.New("writer failed")
	agent := testAgentReturning("Your ticket includes one personal item.")
	out := &failAfterWriter{
		remaining: len("Your ticket includes one personal item."),
		err:       writerErr,
	}

	response, err := agent.StreamResponse(context.Background(), nil, out)
	if !errors.Is(err, writerErr) {
		t.Fatalf("StreamResponse error = %v, want %v", err, writerErr)
	}
	if response != "Your ticket includes one personal item." {
		t.Fatalf("returned response = %q, want unfinished runner response", response)
	}
}

func testAgentReturning(response string) *Agent {
	return NewAgentWithRunnerFunc(func(_ context.Context, _ string, _ []Tool, _ []Message, out io.Writer) (string, error) {
		if _, err := io.WriteString(out, response); err != nil {
			return "", err
		}
		return response, nil
	})
}

type failAfterWriter struct {
	remaining int
	err       error
}

func (w *failAfterWriter) Write(p []byte) (int, error) {
	if len(p) <= w.remaining {
		w.remaining -= len(p)
		return len(p), nil
	}
	n := w.remaining
	w.remaining = 0
	return n, w.err
}