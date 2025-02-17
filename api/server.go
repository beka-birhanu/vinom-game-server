package api

import (
	"context"
	"fmt"

	"github.com/beka-birhanu/vinom-game-server/service/i"
	"github.com/google/uuid"
	grpc "google.golang.org/grpc"
)

type Server struct {
	gameSessionManager i.GameSessionManager

	UnimplementedSessionServer
}

func RegisterNewGamseSessionManager(gsr grpc.ServiceRegistrar, gsm i.GameSessionManager) error {
	server := &Server{
		gameSessionManager: gsm,
	}

	RegisterSessionServer(gsr, server)
	return nil
}

func (s *Server) NewGame(ctx context.Context, r *NewGameRequest) (*NewGameResponse, error) {
	stringIDs := r.GetPlayerIDs()
	parsedIDs := make([]uuid.UUID, 0)
	for _, stringID := range stringIDs {
		id, err := uuid.Parse(stringID)
		if err != nil {
			fmt.Println(err)
			fmt.Println(stringID)
			return nil, err
		}
		parsedIDs = append(parsedIDs, id)
	}

	s.gameSessionManager.NewSession(parsedIDs)
	return &NewGameResponse{}, nil
}

func (s *Server) SessionInfo(ctx context.Context, r *SessionInfoRequest) (*SessionInfoResponse, error) {
	parsedID, err := uuid.Parse(r.GetPlayerID())
	if err != nil {
		return nil, fmt.Errorf("Error parsing playerID: %s", err)
	}

	pubKey, serverAddr, err := s.gameSessionManager.SessionInfo(parsedID)
	return &SessionInfoResponse{
		ServerPubKey: string(pubKey),
		ServerAddr:   serverAddr,
	}, err
}
