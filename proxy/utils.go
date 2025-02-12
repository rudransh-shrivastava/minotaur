package proxy

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

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
			for _, server := range p.servers {
				fmt.Printf("server: %s, avg_response_time: %d, weight: %d\n", server.URL, server.AvgResponseMs, server.Weight)
			}
		}
	}
}
