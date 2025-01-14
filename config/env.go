package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	LoadBalancingMode string
	Servers           []string
	RedisHost         string
	SSLKeyPath        string
	SSLCertPath       string
}

var Envs = initConfig()

func initConfig() Config {
	godotenv.Load()

	loadBalancingMode := getEnv("LOAD_BALANCING_MODE", "")
	if loadBalancingMode != "ROUND_ROBIN" && loadBalancingMode != "WEIGHTED_ROUND_ROBIN" {
		log.Fatalf("Invalid load balancing mode: %s", loadBalancingMode)
	}

	envServers := getEnv("SERVERS", "localhost:8081")
	servers := strings.Split(envServers, ",")

	fmt.Println("Servers:")
	for i, server := range servers {
		fmt.Println("server", i, ":", server)
	}

	return Config{
		Port:              getEnv("PORT", "443"),
		LoadBalancingMode: getEnv("LOAD_BALANCING_MODE", "ROUND_ROBIN"),
		Servers:           servers,
		RedisHost:         getEnv("REDIS_HOST", "localhost:6379"),
		SSLKeyPath:        getEnv("SSL_KEY_PATH", "localhost-key.pem"),
		SSLCertPath:       getEnv("SSL_CERT_PATH", "localhost.pem"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	fmt.Println("Using default value for", key)
	return fallback
}
