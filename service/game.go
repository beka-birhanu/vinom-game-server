package service

import (
	"errors"
	"maps"
	"slices"
	"sync"
	"time"

	game_i "github.com/beka-birhanu/vinom-common/interfaces/game"
	"github.com/google/uuid"
)

// Game-related errors.
var (
	ErrTooManyPlayers        = errors.New("too many players")
	ErrNotEnoughPlayers      = errors.New("not enough players")
	ErrNotBigEnoughDimension = errors.New("dimension is not big enough")
	ErrInvalidPlayerPosition = errors.New("player is out of the maze")
)

// Game constants for configuration and action types.
const (
	moveActionType         = 3 << iota // Action type for movement.
	stateRequestActionType             // Action type for state requests.

	minPlayers = 2 // Minimum number of players.
	maxPlayers = 4 // Maximum number of players.

	minDimension = 3 // Minimum maze dimension (width or height).
)

// Game represents a maze game with players, a maze, and game state.
// It manages player actions, broadcasts game state, and tracks game progress.
type Game struct {
	maze         game_i.Maze                 // The maze structure.
	players      map[uuid.UUID]game_i.Player // Map of players indexed by their IDs.
	version      int64                       // Game state version for synchronization.
	encoder      game_i.GameEncoder          // Encoder for serializing game state.
	stop         chan bool                   // stop channel to signal stop.
	rewardsLeft  int                         // Total rewards left in the maze.
	stateChan    chan []byte                 // Channel for broadcasting state changes.
	actionChan   chan []byte                 // Channel for broadcasting actions.
	endChan      chan []byte                 // Channel to signal game completion.
	Wg           *sync.WaitGroup             // WaitGroup to manage concurrent goroutines.
	sync.RWMutex                             // Read-Write lock for synchronizing access.
}

// NewGame creates a new Game instance with the specified maze, players, and encoder.
// Returns an error if configuration constraints are violated.
func NewGame(maze game_i.Maze, players []game_i.Player, e game_i.GameEncoder) (*Game, error) {
	if len(players) > maxPlayers {
		return nil, ErrTooManyPlayers
	}

	if len(players) < minPlayers {
		return nil, ErrNotEnoughPlayers
	}

	if maze.Width() < minDimension || maze.Height() < minDimension {
		return nil, ErrNotBigEnoughDimension
	}

	playersMap := make(map[uuid.UUID]game_i.Player)
	for _, player := range players {
		if !maze.InBound(int(player.RetrivePos().GetRow()), int(player.RetrivePos().GetCol())) {
			return nil, ErrInvalidPlayerPosition
		}
		playersMap[player.GetID()] = player
		_ = maze.RemoveReward(player.RetrivePos())
	}

	return &Game{
		maze:        maze,
		players:     playersMap,
		rewardsLeft: maze.Width() * maze.Height(),
		encoder:     e,
		stop:        make(chan bool, 1),
		stateChan:   make(chan []byte),
		actionChan:  make(chan []byte),
		endChan:     make(chan []byte),
		Wg:          &sync.WaitGroup{},
	}, nil
}

// Start begins the game and listens for player actions or a timeout.
func (g *Game) Start(gameDuration time.Duration) {
	time.AfterFunc(gameDuration, g.Stop)
	x := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-g.stop:
			close(g.stop)
			return
		case action := <-g.actionChan:
			if len(action) < 2 {
				continue
			}
			g.handleAction(action[0], action[1:])
		case <-x.C:
			g.Wg.Add(1)
			g.broadcastState(false)
		}
	}
}

// handleAction processes incoming actions based on their type.
func (g *Game) handleAction(t byte, move []byte) {
	switch t {
	case stateRequestActionType:
		g.Wg.Add(1)
		go g.broadcastState(false)
	case moveActionType:
		a, err := g.encoder.UnmarshalAction(move)
		if err != nil {
			return
		}
		go g.handleIncomingMove(a)
	}
}

// Stop ends the game, closes channels, and broadcasts the final state.
func (g *Game) Stop() {
	g.stop <- true
	g.Wg.Wait()
	close(g.actionChan)
	close(g.stateChan)
	g.Wg.Add(1)
	g.broadcastState(true)
	close(g.endChan)
}

// broadcastState sends the current game state to all players.
func (g *Game) broadcastState(ended bool) {
	defer g.Wg.Done()
	gameState := g.snapshot()
	gameStatePayload, err := g.encoder.MarshalGameState(gameState)
	if err != nil {
		return
	}

	if ended {
		g.endChan <- gameStatePayload
	} else {
		g.stateChan <- gameStatePayload
	}
}

// snapshot creates a snapshot of the current game state.
func (g *Game) snapshot() game_i.GameState {
	g.RLock()
	defer g.RUnlock()

	gameState := g.encoder.NewGameState()
	gameState.SetVersion(g.version)
	gameState.SetMaze(g.maze)
	gameState.SetPlayers(slices.Collect(maps.Values(g.players)))
	return gameState
}

// handleIncomingMove processes player movement actions.
// It validates the move, updates the state, and broadcasts changes.
func (g *Game) handleIncomingMove(a game_i.Action) {
	g.Lock()
	p, ok := g.players[a.GetID()]
	if !ok {
		g.Unlock()
		return
	}

	curPosition := p.RetrivePos()
	if curPosition.GetRow() != a.RetriveFrom().GetRow() || curPosition.GetCol() != a.RetriveFrom().GetCol() {
		g.Unlock()
		return
	}

	move, err := g.maze.NewValidMove(curPosition, a.GetDirection())
	if err != nil {
		g.Unlock()
		return
	}

	reward, _ := g.maze.Move(move)
	p.SetReward(p.GetReward() + reward)
	p.SetPos(move.To())

	g.version++
	if g.maze.GetTotalReward() == 0 {
		g.Unlock()
		g.Stop()
		return
	}
	g.Unlock()

	g.Wg.Add(1)
	go g.broadcastState(false)
}

// StateChan returns the state change channel.
func (g *Game) StateChan() <-chan []byte {
	return g.stateChan
}

// ActionChan returns the action channel.
func (g *Game) ActionChan() chan<- []byte {
	return g.actionChan
}

// EndChan returns the end channel for the game.
func (g *Game) EndChan() <-chan []byte {
	return g.endChan
}
