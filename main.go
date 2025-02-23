package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/beka-birhanu/udp-socket-manager/crypto"
	udppb "github.com/beka-birhanu/udp-socket-manager/encoding"
	udpsocket "github.com/beka-birhanu/udp-socket-manager/socket"
	gamepb "github.com/beka-birhanu/vinom-common/gameencoder"
	general_i "github.com/beka-birhanu/vinom-common/interfaces/general"
	socket_i "github.com/beka-birhanu/vinom-common/interfaces/socket"
	logger "github.com/beka-birhanu/vinom-common/log"
	"github.com/beka-birhanu/vinom-game-server/api"
	"github.com/beka-birhanu/vinom-game-server/config"
	"github.com/beka-birhanu/vinom-game-server/service"
	"github.com/beka-birhanu/vinom-game-server/service/i"
	maze "github.com/beka-birhanu/wilson-maze"
	"google.golang.org/grpc"
)

// Global variables for dependencies
var (
	grpcConnListener   net.Listener
	grpcServer         *grpc.Server
	udpSocketManager   socket_i.ServerSocketManager
	gameSessionManager i.GameSessionManager
	appLogger          general_i.Logger
)

func initUDPSocketManager() {
	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", "vinom_session_manager", config.Envs.UdpPort))
	if err != nil {
		appLogger.Error(fmt.Sprintf("Resolving server address: %v", err))
		os.Exit(1)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		appLogger.Error(fmt.Sprintf("Generating RSA key: %v", err))
		os.Exit(1)
	}

	rsaEnc := crypto.NewRSA(privateKey)

	serverLogger, err := logger.New("SERVER-SOCKET", config.ColorBlue, os.Stdout)
	if err != nil {
		appLogger.Error(fmt.Sprintf("Creating UDP socket manager logger: %v", err))
		os.Exit(1)
	}
	server, err := udpsocket.NewServerSocketManager(
		udpsocket.ServerConfig{
			ListenAddr:  serverAddr,
			AsymmCrypto: rsaEnc,
			SymmCrypto:  crypto.NewAESCBC(),
			Encoder:     &udppb.Protobuf{},
			HMAC:        &crypto.HMAC{},
			Logger:      serverLogger,
		},
		udpsocket.ServerWithReadBufferSize(config.Envs.UDPBufferSize),
		udpsocket.ServerWithHeartbeatExpiration(3*time.Second),
	)
	if err != nil {
		appLogger.Error(fmt.Sprintf("Creating server UDP socket manager: %v", err))
		os.Exit(1)
	}

	udpSocketManager = server
	appLogger.Info("UDP Socket Manager initialized")
}

func initGameSessionManager() {
	gameLogger, err := logger.New("GAME-MANAGER", config.ColorCyan, os.Stdout)
	if err != nil {
		appLogger.Error(fmt.Sprintf("Creating UDP socket manager logger: %v", err))
		os.Exit(1)
	}
	manager, err := service.NewGameSessionManager(
		&service.Config{
			Socket:       udpSocketManager,
			MazeFactory:  maze.New,
			GameEndcoder: &gamepb.Protobuf{},
			Logger:       gameLogger,
		},
	)
	if err != nil {
		appLogger.Error(fmt.Sprintf("Creating game session manager: %v", err))
		os.Exit(1)
	}
	gameSessionManager = manager
	appLogger.Info("Game Session Manager initialized")
}
func initSessionManagerController() {
	grpcServer = grpc.NewServer()
	err := api.RegisterNewGamseSessionManager(grpcServer, gameSessionManager)
	if err != nil {
		appLogger.Error(fmt.Sprintf("Creating and Registering session manager controller: %v", err))
		os.Exit(1)
	}
	appLogger.Info("Matchmaking controller initialized")
}

// TODO: add socket monitoring.
func main() {
	appLogger, _ = logger.New("APP", config.ColorGreen, os.Stdout)
	initUDPSocketManager()
	initGameSessionManager()
	initSessionManagerController()

	defer func() {
		gameSessionManager.StopAll()
		time.Sleep(2 * time.Second) // Wait for all games to finish sending data. @TODO: use better way to wait, maybe waitgroups.
		udpSocketManager.Stop()
	}()

	go udpSocketManager.Serve()
	appLogger.Info("UDP Socket Manager started serving")

	var err error
	addr := fmt.Sprintf("%s:%v", config.Envs.ProxyIP, config.Envs.GrpcPort)
	grpcConnListener, err = net.Listen("tcp", addr)
	defer func() {
		_ = grpcConnListener.Close()
	}()

	if err != nil {
		appLogger.Error(fmt.Sprintf("Listening tcp: %v", err))
		os.Exit(1)
	}

	appLogger.Info(fmt.Sprintf("Serving gRPC at: %s", addr))

	if err := grpcServer.Serve(grpcConnListener); err != nil {
		appLogger.Error(fmt.Sprintf("Serving gRPC: %v", err))
		os.Exit(1)
	}
}
