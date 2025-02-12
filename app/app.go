package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/rudransh-shrivastava/minotaur/config"
	"github.com/rudransh-shrivastava/minotaur/proxy"
	redisclient "github.com/rudransh-shrivastava/minotaur/redis_client"
	"github.com/rudransh-shrivastava/minotaur/utils"
)

type App struct {
	Ctx context.Context
}

func NewApp(ctx context.Context) *App {
	return &App{Ctx: ctx}
}

func (a *App) Start() {
	configServers := config.Envs.Servers
	var servers []proxy.Server
	for _, server := range configServers {
		servers = append(servers, proxy.Server{
			URL: server,
		})
	}

	redisClient, err := redisclient.NewRedisClient(a.Ctx)
	if err != nil {
		fmt.Println("Error creating redis client", err)
		return
	}

	httpClient := newHttpClient()

	certFile := config.Envs.SSLCertPath
	keyFile := config.Envs.SSLKeyPath

	proxy := proxy.NewProxy(a.Ctx, servers, redisClient, httpClient)

	server := &http.Server{
		Addr:    ":" + config.Envs.Port,
		Handler: http.HandlerFunc(proxy.ProxyHandler),
	}
	if config.Envs.SSLCertPath != "USE_HTTP" {
		fmt.Println("No certificates defined. Using HTTP")
		server = &http.Server{
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		}
	}

	go utils.LogLoop(a.Ctx, &servers)

	if config.Envs.LoadBalancingMode == "WEIGHTED_ROUND_ROBIN" {
		go proxy.StartWeightAdjustment(a.Ctx, 2*time.Second)
	}

	fmt.Printf("Starting reverse proxy on port %s ...\n", config.Envs.Port)
	if config.Envs.SSLCertPath == "USE_HTTP" {
		err = server.ListenAndServe()
		if err != nil {
			fmt.Println("Error starting server:", err)
		}
	} else {
		err = server.ListenAndServeTLS(certFile, keyFile)
		if err != nil {
			fmt.Println("Error starting server:", err)
		}
	}
}

func newHttpClient() *http.Client {
	httpTransport := http.DefaultTransport.(*http.Transport).Clone()
	httpTransport.MaxIdleConns = 1000
	httpTransport.MaxConnsPerHost = 1000
	httpTransport.MaxIdleConnsPerHost = 1000

	httpClient := &http.Client{
		Timeout:   10 * time.Second,
		Transport: httpTransport,
	}
	return httpClient
}
