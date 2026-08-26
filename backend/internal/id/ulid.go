package id

import (
	"fmt"

	"github.com/google/uuid"
)

// NewRequestID generates a time-sortable (K-Sortable) UUIDv7 request identifier.
// Format: req_018f... (RFC 9562 Unix-epoch timestamp millisecond ordered)
// This ensures optimal B-Tree index locality in PostgreSQL and zero fragmentation.
func NewRequestID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return "req_" + uuid.New().String()
	}
	return fmt.Sprintf("req_%s", id.String())
}
