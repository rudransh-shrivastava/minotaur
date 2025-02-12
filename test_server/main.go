package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Server struct {
	Delay         int
	RequestCount  int64
	LastResponse  time.Time
	PatternDelay  bool
	mutex         sync.Mutex
	startupTime   time.Time
	cacheableData map[string][]byte
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	delay := os.Getenv("DELAY")
	intDelay := 0
	if delay != "" {
		var err error
		intDelay, err = strconv.Atoi(delay)
		if err != nil {
			fmt.Println("Error converting delay to int, using 0")
		}
	}

	server := &Server{
		Delay:         intDelay,
		startupTime:   time.Now(),
		cacheableData: make(map[string][]byte),
	}

	// Initialize some cacheable data
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("item-%d", i)
		server.cacheableData[key] = []byte(fmt.Sprintf("cached-content-%d", i))
	}

	http.HandleFunc("/foo", server.fooHandler)                    // Returns "foo"
	http.HandleFunc("/random-delay", server.randomDelayHandler)   // Returns with a random delay between 100-500ms
	http.HandleFunc("/pattern-delay", server.patternDelayHandler) // Alternate between fast and slow responses

	// Cacheable endpoints
	http.HandleFunc("/cached/", server.cacheableHandler) // Returns cached content for 5 minutes
	http.HandleFunc("/dynamic/", server.dynamicHandler)  // Returns dynamic content with varying processing times
	http.HandleFunc("/metrics", server.metricsHandler)   // Server metrics

	fmt.Printf("Starting test backend server on port %s with base delay %d ms\n", port, intDelay)
	http.ListenAndServe(":"+port, nil)
}

func (s *Server) fooHandler(w http.ResponseWriter, r *http.Request) {
	s.incrementRequests()
	if s.Delay > 0 {
		time.Sleep(time.Duration(s.Delay) * time.Millisecond)
	}
	w.Write([]byte("foo"))
}

func (s *Server) randomDelayHandler(w http.ResponseWriter, r *http.Request) {
	s.incrementRequests()
	// Random delay between 100-500ms
	randomDelay := 100 + rand.Intn(400)
	time.Sleep(time.Duration(randomDelay) * time.Millisecond)
	w.Write([]byte(fmt.Sprintf("delayed-%dms", randomDelay)))
}

func (s *Server) patternDelayHandler(w http.ResponseWriter, r *http.Request) {
	s.incrementRequests()
	s.mutex.Lock()
	s.PatternDelay = !s.PatternDelay
	shouldDelay := s.PatternDelay
	s.mutex.Unlock()

	if shouldDelay {
		time.Sleep(300 * time.Millisecond)
		w.Write([]byte("slow-response"))
	} else {
		w.Write([]byte("fast-response"))
	}
}

func (s *Server) cacheableHandler(w http.ResponseWriter, r *http.Request) {
	s.incrementRequests()
	id := r.URL.Path[len("/cached/"):]

	// Set aggressive caching headers
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("ETag", fmt.Sprintf("\"item-%s\"", id))

	if data, exists := s.cacheableData[id]; exists {
		w.Write(data)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) dynamicHandler(w http.ResponseWriter, r *http.Request) {
	s.incrementRequests()
	// Simulate varying processing times
	processingTime := 50 + rand.Intn(200)
	time.Sleep(time.Duration(processingTime) * time.Millisecond)

	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(fmt.Sprintf("dynamic-content-%d", time.Now().UnixNano())))
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	s.mutex.Lock()
	metrics := map[string]interface{}{
		"total_requests": s.RequestCount,
		"uptime_seconds": time.Since(s.startupTime).Seconds(),
		"base_delay_ms":  s.Delay,
	}
	s.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (s *Server) incrementRequests() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.RequestCount++
	s.LastResponse = time.Now()
}
