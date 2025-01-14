package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	redisclient "github.com/rudransh-shrivastava/minotaur/redis_client"
)

type Proxy struct {
	RedisClient     *redisclient.RedisClient
	servers         []Server
	roundRobinIndex uint64
	Context         context.Context
}

type Server struct {
	URL            string
	Count          int
	Weight         int
	AvgResponseMs  int64
	TotalResponses int64
}

func NewProxy(ctx context.Context, servers []Server, redisClient *redisclient.RedisClient) *Proxy {
	for i := range servers {
		// Defaults
		servers[i].Count = 0
		servers[i].Weight = 1
		servers[i].AvgResponseMs = 1
		servers[i].TotalResponses = 0
	}
	return &Proxy{RedisClient: redisClient, servers: servers, Context: ctx}
}

func (p *Proxy) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		p.forwardRequest(w, r)
		return
	}
	cacheKey := r.URL.String()

	cachedResponse, found := p.RedisClient.Get(p.Context, cacheKey)
	if found {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cachedResponse))
		return
	}
	respBody, headers, err := p.forwardRequest(w, r)
	if err != nil {
		http.Error(w, "Error occurred", http.StatusInternalServerError)
		return
	}

	cacheDuration := p.getCacheDuration(headers.Get("Cache-Control"))

	p.RedisClient.Set(p.Context, cacheKey, string(respBody), cacheDuration)

	w.Write(respBody)
}

func (p *Proxy) forwardRequest(w http.ResponseWriter, r *http.Request) ([]byte, http.Header, error) {
	nextServer := p.getNextServer()
	nextServer.Count++
	nextServerURL := nextServer.URL
	serverUrl, err := url.Parse(nextServerURL)
	if err != nil {
		http.Error(w, "Error occurred", http.StatusInternalServerError)
		return nil, nil, err
	}

	r.Host = serverUrl.Host
	r.URL.Host = serverUrl.Host
	r.URL.Scheme = serverUrl.Scheme
	r.RequestURI = ""

	start := time.Now()
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		http.Error(w, "Error occurred", http.StatusInternalServerError)
		return nil, nil, err
	}

	responseTime := time.Since(start).Milliseconds()
	p.updateResponseTime(nextServer, responseTime)

	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		http.Error(w, "Error occurred", http.StatusInternalServerError)
		return nil, nil, err
	}
	headers := response.Header.Clone()
	return respBody, headers, nil
}

func (p *Proxy) getCacheDuration(cacheControl string) time.Duration {
	defaultTTL := 10 * time.Second

	if cacheControl == "" {
		return defaultTTL
	}

	// Parse Cache-Control header
	directives := strings.Split(cacheControl, ",")
	for _, directive := range directives {
		parts := strings.Split(strings.TrimSpace(directive), "=")
		if len(parts) == 2 && parts[0] == "max-age" {
			maxAge, err := strconv.Atoi(parts[1])
			if err == nil {
				return time.Duration(maxAge) * time.Second
			}
		}
	}

	return defaultTTL
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
	}
}

func (p *Proxy) StartWeightAdjustment(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			p.adjustWeightsByResponseTime()
			fmt.Println("--------------------")
			fmt.Println("Adjusted server weights:")
			for _, server := range p.servers {
				fmt.Printf("server: %s, avg_response_time: %d, weight: %d\n", server.URL, server.AvgResponseMs, server.Weight)
			}
		}
	}()
}
