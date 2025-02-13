package proxy

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func NewHttpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true, // If clients accept compressed responses
		},
		Timeout: 30 * time.Second,
	}
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
	p.serverLock.RLock()
	defer p.serverLock.RUnlock()

	var bestServer *Server
	var bestScore float64 = -1

	now := time.Now()

	for i := range p.servers {
		server := &p.servers[i]

		// Skip servers that failed recently
		if now.Sub(server.LastCheck) < 10*time.Second && server.AvgResponseMs > 5000 {
			continue
		}

		// Score based on weight and response time
		score := float64(server.Weight) / (float64(server.AvgResponseMs) + 1)
		if score > bestScore {
			bestScore = score
			bestServer = server
		}
	}

	if bestServer == nil {
		return &p.servers[0] // Fallback
	}

	return bestServer
}

func (p *Proxy) updateResponseTime(server *Server, responseTime int64) {
	const alpha = 0.5 // Smoothing factor for Exponential Moving Average (EMA)
	if server.TotalResponses == 0 {
		// Init
		server.AvgResponseMs = responseTime
	} else {
		server.AvgResponseMs = int64(float64(server.AvgResponseMs)*(1-alpha) + float64(responseTime)*alpha)
	}
	server.TotalResponses++
}

func (p *Proxy) adjustWeightsByResponseTime() {
	const smoothingFactor = 50 // Add to all response times for fairness
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

func (p *Proxy) StartWeightAdjustment(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.adjustWeightsByResponseTime()
			fmt.Println("--------------------")
			fmt.Println("Adjusted server weights:")
			for i := range p.servers {
				fmt.Printf("server: %s, avg_response_time: %d, weight: %d\n", p.servers[i].URL, p.servers[i].AvgResponseMs, p.servers[i].Weight)
			}
		}
	}
}
