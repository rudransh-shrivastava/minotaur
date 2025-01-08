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
	http.HandleFunc("/foo", fooHandler)
	http.HandleFunc("/foo/bar", fooBarHandler)
	fmt.Println("running server")
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
	fmt.Println("foo handler called at", time.Now())
	w.Write([]byte("foo"))
}

func fooBarHandler(w http.ResponseWriter, r *http.Request) {
	delay := os.Getenv("DELAY")
	if delay == "" {
		delay = "1000"
	}
	intDelay, err := strconv.Atoi(delay)
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Duration(intDelay) * time.Millisecond)
	fmt.Println("foo bar handler called at", time.Now())
	w.Write([]byte("foo bar"))
}
