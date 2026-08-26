package id

import (
	"strings"
	"testing"
	"time"
)

func TestNewRequestID_Format(t *testing.T) {
	id := NewRequestID()
	if !strings.HasPrefix(id, "req_") {
		t.Errorf("expected request ID to start with req_, got %s", id)
	}

	if len(id) < 30 {
		t.Errorf("expected standard length UUIDv7 ID, got %d chars", len(id))
	}
}

func TestNewRequestID_TimeOrdering(t *testing.T) {
	id1 := NewRequestID()
	time.Sleep(2 * time.Millisecond)
	id2 := NewRequestID()

	// In UUIDv7, lexicographical comparison matches timestamp order
	if id1 >= id2 {
		t.Errorf("expected id1 < id2 for time-sortable IDs: %s >= %s", id1, id2)
	}
}
