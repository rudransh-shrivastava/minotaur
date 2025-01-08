package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func main() {
	target := "http://localhost:8081"
	backendUrl, err := url.Parse(target)
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleReverseProxy(w, r, backendUrl)
	})
	fmt.Println("reverse proxy running on port 8080")
	http.ListenAndServe(":8080", nil)
}

func handleReverseProxy(w http.ResponseWriter, r *http.Request, backendUrl *url.URL) {
	r.Host = backendUrl.Host
	r.URL.Host = backendUrl.Host
	r.URL.Scheme = backendUrl.Scheme
	r.RequestURI = ""
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		http.Error(w, "Error occurred", http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()
	w.WriteHeader(http.StatusOK)
	io.Copy(w, response.Body)
}
