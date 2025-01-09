package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	delay := os.Getenv("DELAY")
	if delay == "" {
		delay = "1000"
	}
	http.HandleFunc("/foo", fooHandler)
	fmt.Printf("starting a test backend server on port %s with delay %s", port, delay)
	http.ListenAndServe(":"+port, nil)
}

func fooHandler(w http.ResponseWriter, r *http.Request) {
	delay := os.Getenv("DELAY")
	if delay == "" {
		delay = "1000"
	}
	intDelay, err := strconv.Atoi(delay)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Duration(intDelay) * time.Millisecond)
	fmt.Println("foo API called at", time.Now())
	w.Write([]byte("foo"))
}
