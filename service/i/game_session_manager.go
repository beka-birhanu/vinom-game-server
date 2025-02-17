package i

import (
	"github.com/google/uuid"
)

// GameSessionManager manages game sessions and provides session-related information.
type GameSessionManager interface {
	// NewSession initializes a new game session with the given player IDs.
	NewSession([]uuid.UUID)

	StopAll()

	// SessionInfo returns the public key, socket address.
	SessionInfo(uuid.UUID) ([]byte, string, error)
}
