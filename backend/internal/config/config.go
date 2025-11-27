package config

import (
	"os"
)

type Config struct {
	Port        string
	Neo4jURI    string
	Neo4jUser   string
	Neo4jPass   string
	TEI_URL     string
	ReposPath   string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("BACKEND_PORT", "3001"),
		Neo4jURI:    getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUser:   getEnv("NEO4J_USER", "neo4j"),
		Neo4jPass:   getEnv("NEO4J_PASSWORD", "neograph_password"),
		TEI_URL:     getEnv("TEI_URL", "http://localhost:8080"),
		ReposPath:   getEnv("REPOS_PATH", "./repos"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
