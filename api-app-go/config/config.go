package config

import (
	"os"
	"strconv"
)

// Config holds all runtime configuration. Values are read from environment
// variables; defaults mirror api-app/src/main/resources/application.properties.
type Config struct {
	// IBM MQ connection
	ConnectionList string // IBM_MQ_CONNECTION_LIST
	Host           string // IBM_MQ_HOST
	Port           int    // IBM_MQ_PORT
	Channel        string // IBM_MQ_CHANNEL
	QueueManager   string // IBM_MQ_QUEUE_MANAGER
	Username       string // IBM_MQ_USERNAME
	Password       string // IBM_MQ_PASSWORD
	Queue          string // IBM_MQ_QUEUE

	// HTTP server
	ServerPort string // SERVER_PORT
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// Load reads configuration from environment variables with defaults.
func Load() *Config {
	return &Config{
		ConnectionList: getenv("IBM_MQ_CONNECTION_LIST", "localhost(1414),localhost(1415),localhost(1416)"),
		Host:           getenv("IBM_MQ_HOST", "localhost"),
		Port:           getenvInt("IBM_MQ_PORT", 1414),
		Channel:        getenv("IBM_MQ_CHANNEL", "DEV.APP.SVRCONN"),
		QueueManager:   getenv("IBM_MQ_QUEUE_MANAGER", "QM1"),
		Username:       getenv("IBM_MQ_USERNAME", "app"),
		Password:       getenv("IBM_MQ_PASSWORD", "passw0rd"),
		Queue:          getenv("IBM_MQ_QUEUE", "DEV.DEMO.QL.IN"),
		ServerPort:     getenv("SERVER_PORT", "8081"),
	}
}
