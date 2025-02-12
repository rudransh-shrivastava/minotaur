package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/cespare/xxhash/v2"
	redisclient "github.com/rudransh-shrivastava/minotaur/redis_client"
)

type Proxy struct {
	RedisClient     *redisclient.RedisClient
	HttpClient      *http.Client
	servers         []Server
	serverLock      sync.RWMutex
	reqGroup        singleflight.Group
	hashFunc        *xxhash.Digest
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
	if r.Method != http.MethodGet {
		p.forwardRequest(w, r)
		return
	}
	cacheKey := r.URL.String()
	cacheTimeNow := time.Now()
	cachedResponse, found := p.RedisClient.Get(p.Context, cacheKey)
	if found {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cachedResponse))
		return
	}
	fmt.Println("Time taken to miss reponse from cache: ", time.Since(cacheTimeNow))
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
	getNextServerNow := time.Now()
	nextServer := p.getNextServer()
	fmt.Println("Time taken to get next server: ", time.Since(getNextServerNow))
	nextServer.Count++
	nextServerURL := nextServer.URL
	fmt.Println("Routing the request to server: ", nextServerURL)

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
	requestTimeNow := time.Now()
	response, err := p.HttpClient.Do(r)
	fmt.Println("Time taken for the request to backend server: ", time.Since(requestTimeNow))
	if err != nil {
		http.Error(w, "Error occurred", http.StatusInternalServerError)
		return nil, nil, err
	}

	responseTime := time.Since(start).Milliseconds()
	updateReponseTimeNow := time.Now()
	p.updateResponseTime(nextServer, responseTime)
	fmt.Println("Time taken to update reponse time: ", time.Since(updateReponseTimeNow))

	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		http.Error(w, "Error occurred", http.StatusInternalServerError)
		return nil, nil, err
	}
	headers := response.Header.Clone()
	fmt.Println("Time taken for the whole opertaion: ", time.Since(getNextServerNow))
	return respBody, headers, nil
}
