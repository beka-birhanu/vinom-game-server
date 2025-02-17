package i

import (
	"time"
)

// GameServer defines the interface for a maze game service.
type GameServer interface {
	// Start begins the game and listens for player actions or a timeout.
	Start(gameDuration time.Duration)

	// Stop ends the game, closes channels, and broadcasts the final state.
	Stop()

	// StateChan returns the state change channel.
	StateChan() <-chan []byte

	// ActionChan returns the action channel.
	ActionChan() chan<- []byte

	// EndChan returns the end channel for the game.
	EndChan() <-chan []byte
}
