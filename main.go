package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rudransh-shrivastava/minotaur/proxy"
)

func main() {
	servers := []proxy.Server{
		{
			URL: "http://localhost:8081",
		},
		{
			URL: "http://localhost:8082",
		},
		{
			URL: "http://localhost:8083",
		},
	}
	proxy := proxy.NewProxy(servers)

	http.HandleFunc("/", proxy.ProxyHandler)

	go logLoop(&servers)
	go proxy.StartWeightAdjustment(2 * time.Second)

	fmt.Println("reverse proxy running on port 8080")
	http.ListenAndServe(":8080", nil)
}

func logLoop(servers *[]proxy.Server) {
	for {
		time.Sleep(10 * time.Second)
		fmt.Println("server status")
		for _, server := range *servers {
			fmt.Printf("server: %s, count: %d\n", server.URL, server.Count)
		}
	}
}
