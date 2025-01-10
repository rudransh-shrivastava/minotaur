package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Server struct {
	Delay int
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	delay := os.Getenv("DELAY")
	intDelay := 0
	if delay == "" {
		intDelay = 0
	}
	intDelay, err := strconv.Atoi(delay)
	if err != nil {
		fmt.Println("Error converting delay to int")
	}
	server := Server{Delay: intDelay}

	http.HandleFunc("/foo", server.fooHandler)

	fmt.Printf("starting a test backend server on port %s with delay %s", port, delay)
	http.ListenAndServe(":"+port, nil)
}

func (s *Server) fooHandler(w http.ResponseWriter, r *http.Request) {
	if s.Delay > 0 {
		time.Sleep(time.Duration(s.Delay) * time.Millisecond)
	}
	w.Write([]byte("foo"))
}
