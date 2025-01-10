package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
)

type Proxy struct {
	servers         []Server
	roundRobinIndex uint64
}

type Server struct {
	URL            string
	Count          int
	Weight         int
	AvgResponseMs  int64
	TotalResponses int64
}

func NewProxy(servers []Server) *Proxy {
	for i := range servers {
		// Defaults
		servers[i].Count = 0
		servers[i].TotalResponses = 0
		servers[i].Weight = 1
		servers[i].AvgResponseMs = 1
	}
	return &Proxy{servers: servers}
}

func (p *Proxy) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	nextServer := p.getNextServer()
	nextServer.Count++
	nextServerURL := nextServer.URL
	serverUrl, err := url.Parse(nextServerURL)
	if err != nil {
		http.Error(w, "Error occurred", http.StatusInternalServerError)
		return
	}

	r.Host = serverUrl.Host
	r.URL.Host = serverUrl.Host
	r.URL.Scheme = serverUrl.Scheme
	r.RequestURI = ""

	start := time.Now()
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		http.Error(w, "Error occurred", http.StatusInternalServerError)
		return
	}
	responseTime := time.Since(start).Milliseconds()
	p.updateResponseTime(nextServer, responseTime)

	defer response.Body.Close()
	w.WriteHeader(http.StatusOK)
	io.Copy(w, response.Body)
}

func (p *Proxy) getNextServer() *Server {
	totalWeight := 0
	for _, server := range p.servers {
		totalWeight += server.Weight
	}

	index := int(atomic.AddUint64(&p.roundRobinIndex, 1)) % totalWeight
	for i := range p.servers {
		if index < p.servers[i].Weight {
			return &p.servers[i]
		}
		index -= p.servers[i].Weight
	}

	return &p.servers[0] // Fallback, should never reach here
}

func (p *Proxy) updateResponseTime(server *Server, responseTime int64) {
	const alpha = 0.8 // Smoothing factor for Exponential Moving Average (EMA)
	if server.TotalResponses == 0 {
		// Init
		server.AvgResponseMs = responseTime
	} else {
		server.AvgResponseMs = int64(float64(server.AvgResponseMs)*(1-alpha) + float64(responseTime)*alpha)
	}
	server.TotalResponses++
}

func (p *Proxy) adjustWeightsByResponseTime() {
	const smoothingFactor = 1 // Add 1ms to all response times for fairness
	for i := range p.servers {
		server := &p.servers[i]
		if server.AvgResponseMs == 0 {
			server.AvgResponseMs = 1 // Divide by zero prevention
		}
		server.Weight = int(1000 / (server.AvgResponseMs + smoothingFactor))
		if server.Weight < 1 {
			server.Weight = 1
		}
		fmt.Printf("Server: %s, AvgResponseMs: %d, Weight: %d\n", server.URL, server.AvgResponseMs, server.Weight)
	}
}

func (p *Proxy) StartWeightAdjustment(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			p.adjustWeightsByResponseTime()
			fmt.Println("Adjusted server weights:")
			for _, server := range p.servers {
				fmt.Printf("Server: %s, AvgResponseMs: %d, Weight: %d\n", server.URL, server.AvgResponseMs, server.Weight)
			}
		}
	}()
}
