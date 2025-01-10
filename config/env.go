package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	LoadBalancingMode string
	Servers           []string
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
	fmt.Println(servers)
	return Config{
		Port:              getEnv("PORT", "8080"),
		LoadBalancingMode: getEnv("LOAD_BALANCING_MODE", "ROUND_ROBIN"),
		Servers:           servers,
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		v, err := strconv.Atoi(value)
		if err != nil {
			log.Fatalf("environment variable %s could not be converted to type int \n error: %v", key, err)
		}
		return v
	}
	return fallback
}
