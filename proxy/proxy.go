package proxy

import (
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
)

type Proxy struct {
	servers         []string
	roundRobinIndex uint64
}

func NewProxy(servers []string) *Proxy {
	return &Proxy{servers: servers}
}

func (p *Proxy) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	atomic.AddUint64(&p.roundRobinIndex, 1)

	server := p.servers[p.roundRobinIndex%uint64(len(p.servers))]
	serverUrl, err := url.Parse(server)
	if err != nil {
		http.Error(w, "Error occurred", http.StatusInternalServerError)
		return
	}

	r.Host = serverUrl.Host
	r.URL.Host = serverUrl.Host
	r.URL.Scheme = serverUrl.Scheme
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
