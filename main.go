package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rudransh-shrivastava/minotaur/config"
	"github.com/rudransh-shrivastava/minotaur/proxy"
	"github.com/rudransh-shrivastava/minotaur/utils"
)

func main() {
	configServers := config.Envs.Servers
	var servers []proxy.Server
	for _, server := range configServers {
		servers = append(servers, proxy.Server{
			URL: server,
		})
	}

	proxy := proxy.NewProxy(servers)

	http.HandleFunc("/", proxy.ProxyHandler)

	go utils.LogLoop(&servers)

	if config.Envs.LoadBalancingMode == "WEIGHTED_ROUND_ROBIN" {
		go proxy.StartWeightAdjustment(2 * time.Second)
	}

	fmt.Println("reverse proxy running on port 8080")
	http.ListenAndServe(":8080", nil)
}
