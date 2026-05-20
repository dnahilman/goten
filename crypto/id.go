package crypto

import "github.com/google/uuid"

const Prefix = "g10_"

// NewID returns a time-sortable ID with g10_ prefix, e.g. "g10_018f4a23-1234-7890-abcd-ef1234567890".
func NewID() string {
	return Prefix + uuid.Must(uuid.NewV7()).String()
}
