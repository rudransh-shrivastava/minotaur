package main

import (
	"context"
	"crypto/tls"
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

	certFile := config.Envs.SSLCertPath
	keyFile := config.Envs.SSLKeyPath

	proxy := proxy.NewProxy(ctx, servers, redisClient)

	server := &http.Server{
		Addr:    ":" + config.Envs.Port,
		Handler: http.HandlerFunc(proxy.ProxyHandler),
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	go utils.LogLoop(&servers)

	if config.Envs.LoadBalancingMode == "WEIGHTED_ROUND_ROBIN" {
		go proxy.StartWeightAdjustment(2 * time.Second)
	}

	fmt.Printf("Starting HTTPS reverse proxy on port %s ...\n", config.Envs.Port)
	err = server.ListenAndServeTLS(certFile, keyFile)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
