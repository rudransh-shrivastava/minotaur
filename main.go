package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rudransh-shrivastava/minotaur/config"
	"github.com/rudransh-shrivastava/minotaur/proxy"
	redisclient "github.com/rudransh-shrivastava/minotaur/redis_client"
	"github.com/rudransh-shrivastava/minotaur/utils"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		fmt.Println("Shutting down...")
		cancel()
		os.Exit(0)
	}()

	configServers := config.Envs.Servers
	var servers []proxy.Server
	for _, server := range configServers {
		servers = append(servers, proxy.Server{
			URL: server,
		})
	}

	redisClient, err := redisclient.NewRedisClient(ctx)
	if err != nil {
		fmt.Println("Error creating redis client", err)
	}

	proxy := proxy.NewProxy(ctx, servers, redisClient)

	http.HandleFunc("/", proxy.ProxyHandler)

	go utils.LogLoop(&servers)

	if config.Envs.LoadBalancingMode == "WEIGHTED_ROUND_ROBIN" {
		go proxy.StartWeightAdjustment(2 * time.Second)
	}

	fmt.Println("reverse proxy running on port 8080")
	http.ListenAndServe(":8080", nil)
}
