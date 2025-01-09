package main

import (
	"fmt"
	"net/http"

	"github.com/rudransh-shrivastava/minotaur/proxy"
)

func main() {
	servers := []string{"http://localhost:8081", "http://localhost:8082", "http://localhost:8083"}

	proxy := proxy.NewProxy(servers)

	http.HandleFunc("/", proxy.ProxyHandler)

	fmt.Println("reverse proxy running on port 8080")
	http.ListenAndServe(":8080", nil)
}
