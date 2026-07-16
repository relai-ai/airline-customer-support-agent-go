package airline

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var sessionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,80}$`)

// Message is a single chat message stored in a session log.
type Message struct {
	Role      string
	Content   string
	CreatedAt string
}

// Session summarizes a stored chat session log.
type Session struct {
	ID           string
	CreatedAt    string
	UpdatedAt    string
	MessageCount int
	Preview      string
}

// On-disk JSONL records. Field order and snake_case keys are kept
// byte-compatible with the Python and TypeScript versions so the logs work
// with RELAI log-based learning environments.
type sessionRecord struct {
	Type      string `json:"type"`
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}

type messageRecord struct {
	Type      string `json:"type"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// LogsDir returns the absolute directory that holds session logs.
func LogsDir() (string, error) {
	dir := os.Getenv("AIRLINE_SUPPORT_LOG_DIR")
	if dir == "" {
		dir = "logs"
	}
	return filepath.Abs(dir)
}

func ensureLogsDir() (string, error) {
	dir, err := LogsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// ValidateSessionID rejects ids that cannot be used as a safe file name.
func ValidateSessionID(sessionID string) (string, error) {
	if !sessionIDPattern.MatchString(sessionID) {
		return "", fmt.Errorf("session id contains unsupported characters")
	}
	return sessionID, nil
}

// SessionPath returns the log file path for a session id.
func SessionPath(sessionID string) (string, error) {
	if _, err := ValidateSessionID(sessionID); err != nil {
		return "", err
	}
	dir, err := ensureLogsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, sessionID+".jsonl"), nil
}

// SessionIDFromLogName derives a session id from a log file name.
func SessionIDFromLogName(logName string) (string, error) {
	name := filepath.Base(logName)
	name = strings.TrimSuffix(name, ".jsonl")
	return ValidateSessionID(name)
}

func newSessionID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "session-" + hex.EncodeToString(buf), nil
}

// CreateSession creates a new log file and writes the session header record.
func CreateSession(sessionID string, overwrite bool) (Session, error) {
	if sessionID == "" {
		var err error
		if sessionID, err = newSessionID(); err != nil {
			return Session{}, err
		}
	} else if _, err := ValidateSessionID(sessionID); err != nil {
		return Session{}, err
	}

	path, err := SessionPath(sessionID)
	if err != nil {
		return Session{}, err
	}

	flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if !overwrite {
		flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return Session{}, err
	}
	defer file.Close()

	timestamp := utcNow()
	if err := writeJSONLine(file, sessionRecord{Type: "session", ID: sessionID, CreatedAt: timestamp}); err != nil {
		return Session{}, err
	}

	return Session{
		ID:           sessionID,
		CreatedAt:    timestamp,
		UpdatedAt:    timestamp,
		MessageCount: 0,
		Preview:      "New session",
	}, nil
}

// DeleteSession removes a session log. It returns false (no error) when the
// session does not exist.
func DeleteSession(sessionID string) (bool, error) {
	path, err := SessionPath(sessionID)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	}
	if err := os.Remove(path); err != nil {
		return false, err
	}
	return true, nil
}

// AppendMessage appends a user or assistant message to a session log.
func AppendMessage(sessionID, role, content string) (Message, error) {
	if role != "user" && role != "assistant" {
		return Message{}, fmt.Errorf("role must be user or assistant")
	}
	path, err := SessionPath(sessionID)
	if err != nil {
		return Message{}, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return Message{}, err
	}
	defer file.Close()

	message := Message{Role: role, Content: content, CreatedAt: utcNow()}
	record := messageRecord{Type: "message", Role: message.Role, Content: message.Content, CreatedAt: message.CreatedAt}
	if err := writeJSONLine(file, record); err != nil {
		return Message{}, err
	}
	return message, nil
}

// ReadMessages returns the ordered chat messages stored in a session log.
func ReadMessages(sessionID string) ([]Message, error) {
	path, err := SessionPath(sessionID)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var messages []Message
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record messageRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, err
		}
		if record.Type != "message" {
			continue
		}
		messages = append(messages, Message{
			Role:      record.Role,
			Content:   record.Content,
			CreatedAt: record.CreatedAt,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

// ListSessions returns stored sessions, most recently updated first.
func ListSessions() ([]Session, error) {
	dir, err := ensureLogsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	type fileInfo struct {
		id      string
		modTime time.Time
	}
	var files []fileInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		files = append(files, fileInfo{
			id:      strings.TrimSuffix(entry.Name(), ".jsonl"),
			modTime: info.ModTime(),
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	var sessions []Session
	for _, f := range files {
		messages, err := ReadMessages(f.id)
		if err != nil {
			return nil, err
		}
		createdAt := f.modTime.UTC().Format(time.RFC3339Nano)
		updatedAt := createdAt
		if len(messages) > 0 {
			createdAt = messages[0].CreatedAt
			updatedAt = messages[len(messages)-1].CreatedAt
		}
		preview := "New session"
		for _, m := range messages {
			if m.Role == "user" {
				preview = m.Content
				break
			}
		}
		if len(preview) > 96 {
			preview = preview[:96]
		}
		sessions = append(sessions, Session{
			ID:           f.id,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
			MessageCount: len(messages),
			Preview:      preview,
		})
	}
	return sessions, nil
}

func writeJSONLine(file *os.File, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}
