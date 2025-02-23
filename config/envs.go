package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds the application's configuration values.
type Config struct {
	ProxyIP  string // Proxy IP of client server
	HostIP   string // Host IP of server
	GrpcPort int    // Port for the GRPC server

	UdpPort                int // Port for the UDP socket
	UDPBufferSize          int // Size of the buffer for incoming UDP packets (in bytes)
	UDPHeartbeatExpiration int // Expiration time for UDP heartbeat (in milliseconds)
}

// Envs holds the application's configuration loaded from environment variables.
var Envs = initConfig()

// initConfig initializes and returns the application configuration.
// It loads environment variables from a .env file.
func initConfig() Config {
	// Load .env file if available
	if err := godotenv.Load(); err != nil {
		log.Printf("[APP] [INFO] .env file not found or could not be loaded: %v", err)
	}

	// Populate the Config struct with required environment variables
	return Config{
		ProxyIP:  mustGetEnv("PROXY_IP"),
		HostIP:   mustGetEnv("Host_IP"),
		GrpcPort: mustGetEnvAsInt("GRPC_PORT"),

		UdpPort:                mustGetEnvAsInt("UDP_PORT"),
		UDPBufferSize:          mustGetEnvAsInt("UDP_BUFFER_SIZE"),
		UDPHeartbeatExpiration: mustGetEnvAsInt("UDP_HEARTBEAT_EXPIRATION"),
	}
}

// mustGetEnv retrieves the value of an environment variable or logs a fatal error if not set.
func mustGetEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		log.Fatalf("%s[APP]%s %s[FATAL]%s Environment variable %s is not set", ColorGreen, ColorReset, ColorRed, ColorReset, key)
	}
	return value
}

// mustGetEnvAsInt retrieves the value of an environment variable as an integer or logs a fatal error if not set or cannot be parsed.
func mustGetEnvAsInt(key string) int {
	valueStr := mustGetEnv(key)
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Fatalf("[APP] [FATAL] Environment variable %s must be an integer: %v", key, err)
	}
	return value
}
