package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/test", handler)
	fmt.Println("running server")
	http.ListenAndServe(":8081", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("received a request", time.Now())
	w.Write([]byte("Hello, World!"))
}
