package proxy

import (
	"log"
	"net/http"
	"sync/atomic"
)

type LoadBalancer struct {
	backends []string
	counter  uint64
}

// NewLoadBalancer initializes a load balancer with backends
func NewLoadBalancer(backends []string) *LoadBalancer {
	return &LoadBalancer{backends: backends}
}

// NextBackend returns the next backend using round-robin strategy
func (lb *LoadBalancer) NextBackend() string {
	if len(lb.backends) == 0 {
		log.Println("No backends available")
		return "" // or handle error gracefully
	}
	index := atomic.AddUint64(&lb.counter, 1) % uint64(len(lb.backends))
	return lb.backends[index]
}

// ServeHTTP forwards requests to the selected backend
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target := lb.NextBackend()
	ReverseProxy(target).ServeHTTP(w, r)
}
