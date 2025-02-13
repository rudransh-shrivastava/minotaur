package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	redisclient "github.com/rudransh-shrivastava/minotaur/redis_client"
)

type Proxy struct {
	RedisClient     *redisclient.RedisClient
	HttpClient      *http.Client
	servers         []Server
	serverLock      sync.RWMutex
	roundRobinIndex uint64
	Context         context.Context
}

type Server struct {
	URL            string
	Count          int
	Weight         int
	AvgResponseMs  int64
	TotalResponses int64
	LastCheck      time.Time
	mutex          sync.Mutex
}

func NewProxy(ctx context.Context, servers []Server, redisClient *redisclient.RedisClient, httpClient *http.Client) *Proxy {
	for i := range servers {
		// Defaults
		servers[i].Count = 0
		servers[i].Weight = 1
		servers[i].AvgResponseMs = 1
		servers[i].TotalResponses = 0
	}
	return &Proxy{RedisClient: redisClient, HttpClient: httpClient, servers: servers, Context: ctx}
}

func (p *Proxy) ProxyHandler(w http.ResponseWriter, r *http.Request) {

	// Check if the method is not GET, that means we cannot
	// cache it. Also, if the request has Cache-Control: no-cache
	// we should not cache it.
	if r.Method != http.MethodGet || r.Header.Get("Cache-Control") == "no-cache" {
		respBody, headers, err := p.forwardRequest(r)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			http.Error(w, "Error occurred", http.StatusInternalServerError)
			return
		}
		// Copy over the headers
		for k, v := range headers {
			w.Header()[k] = v
		}
		w.Write(respBody)
		return
	}

	// Find cached response
	cacheKey := r.URL.String()
	cachedResponse, found := p.RedisClient.Get(p.Context, cacheKey)
	if found {
		w.Write([]byte(cachedResponse))
		return
	}
	// Cached response not found, forward request
	respBody, headers, err := p.forwardRequest(r)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		http.Error(w, "Error occurred", http.StatusInternalServerError)
		return
	}

	cacheDuration := p.getCacheDuration(headers.Get("Cache-Control"))
	p.RedisClient.Set(p.Context, cacheKey, string(respBody), cacheDuration)
	// Copy headers before writing body
	for k, v := range headers {
		w.Header()[k] = v
	}
	w.Write(respBody)
}

func (p *Proxy) forwardRequest(r *http.Request) ([]byte, http.Header, error) {
	nextServer := p.getNextServer()
	nextServer.Count++
	nextServerURL := nextServer.URL
	fmt.Println("Routing the request to server: ", nextServerURL)

	serverUrl, err := url.Parse(nextServerURL)
	if err != nil {
		return nil, nil, err
	}

	r.Host = serverUrl.Host
	r.URL.Host = serverUrl.Host
	r.URL.Scheme = serverUrl.Scheme
	r.RequestURI = ""

	start := time.Now()
	response, err := p.HttpClient.Do(r)
	if err != nil {
		return nil, nil, err
	}

	responseTime := time.Since(start).Milliseconds()
	p.updateResponseTime(nextServer, responseTime)

	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, nil, err
	}
	headers := response.Header.Clone()
	return respBody, headers, nil
}
