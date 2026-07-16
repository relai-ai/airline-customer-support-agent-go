package airline

import (
	"strings"
	"testing"
)

func TestSessionMessagesAreLogged(t *testing.T) {
	t.Setenv("AIRLINE_SUPPORT_LOG_DIR", t.TempDir())

	session, err := CreateSession("", false)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if _, err := AppendMessage(session.ID, "user", "Can I change my seat for SKY123?"); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	if _, err := AppendMessage(session.ID, "assistant", "I can help after verifying the booking."); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	messages, err := ReadMessages(session.ID)
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(messages) != 2 || messages[0].Role != "user" || messages[1].Role != "assistant" {
		t.Fatalf("messages = %+v, want [user, assistant]", messages)
	}
	if messages[0].Content != "Can I change my seat for SKY123?" {
		t.Errorf("first content = %q", messages[0].Content)
	}

	sessions, err := ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) == 0 || sessions[0].ID != session.ID {
		t.Fatalf("sessions[0] = %+v, want id %s", sessions, session.ID)
	}
	if sessions[0].MessageCount != 2 {
		t.Errorf("message count = %d, want 2", sessions[0].MessageCount)
	}
	if !strings.Contains(sessions[0].Preview, "change my seat") {
		t.Errorf("preview = %q", sessions[0].Preview)
	}
}

func TestDeleteSessionRemovesIt(t *testing.T) {
	t.Setenv("AIRLINE_SUPPORT_LOG_DIR", t.TempDir())

	session, err := CreateSession("", false)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	deleted, err := DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	if !deleted {
		t.Error("DeleteSession = false, want true")
	}

	// Deleting a missing session is a no-op (false), not an error.
	deleted, err = DeleteSession(session.ID)
	if err != nil {
		t.Fatalf("DeleteSession (missing): %v", err)
	}
	if deleted {
		t.Error("DeleteSession (missing) = true, want false")
	}
}

func TestCreateSessionRejectsBadID(t *testing.T) {
	t.Setenv("AIRLINE_SUPPORT_LOG_DIR", t.TempDir())

	if _, err := CreateSession("bad id!", false); err == nil {
		t.Fatal("expected error for invalid session id")
	}
}

func TestSessionIDFromLogName(t *testing.T) {
	t.Setenv("AIRLINE_SUPPORT_LOG_DIR", t.TempDir())

	id, err := SessionIDFromLogName("logs/off-topic-guardrail.jsonl")
	if err != nil {
		t.Fatalf("SessionIDFromLogName: %v", err)
	}
	if id != "off-topic-guardrail" {
		t.Errorf("id = %q, want off-topic-guardrail", id)
	}
}
