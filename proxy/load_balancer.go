package proxy

import (
	
	"log"
	"hash/fnv"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type LoadBalancer struct {
	backends    []string
	counter     uint64
	healthCheck map[string]*atomic.Bool
	strategy   string
	connCounts map[string]*int64
	mu         sync.Mutex
}

// NewLoadBalancer initializes a load balancer with backends
func NewLoadBalancer(backends []string, strategy string) *LoadBalancer {
	var urls []string
	healthCheck := make(map[string]*atomic.Bool)

	for _, backend := range backends {
		urls = append(urls, backend)
		healthCheck[backend] = &atomic.Bool{}
		healthCheck[backend].Store(true) // Assume initially healthy
	}

	lb := &LoadBalancer{
		backends:    urls,
		healthCheck: healthCheck,
		strategy:   strategy,
		connCounts:  make(map[string]*int64),
	}

	for _, backend := range urls {
		var count int64
		lb.connCounts[backend] = &count
	}

	// Start periodic health checks
	go lb.startHealthCheck()

	return lb
}

// startHealthCheck continuously checks backend health
func (lb *LoadBalancer) startHealthCheck() {
	for {
		for _, backend := range lb.backends {
			backendURL := backend
			client := &http.Client{Timeout: 2 * time.Second} // Short timeout for health check

			resp, err := client.Get(backendURL)
			if err != nil || resp.StatusCode >= 500 {
				if lb.healthCheck[backendURL].Load() { // Log only on state change
					log.Printf("[HealthCheck] Backend %s is DOWN\n", backendURL)
				}
				lb.healthCheck[backendURL].Store(false)
			} else {
				if !lb.healthCheck[backendURL].Load() {
					log.Printf("[HealthCheck] Backend %s is UP\n", backendURL)
				}
				lb.healthCheck[backendURL].Store(true)
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
		time.Sleep(10 * time.Second) // Health check interval
	}
}

// NextBackend returns the next backend using round-robin strategy
func (lb *LoadBalancer) NextBackend(r *http.Request) string {
	totalBackends := len(lb.backends)
	if totalBackends == 0 {
		log.Println("No backends available")
		return "" // or handle error gracefully
	}

	switch lb.strategy {
	case "least-connections":
		return lb.LeastConnectionsBackend()
	case "ip-hash":
		return lb.IPHashBackend(r)
	default:
		return lb.RoundRobinBackend()
	}
	return "" // No healthy backends available
}

// RoundRobinBackend selects backends in order
func (lb *LoadBalancer) RoundRobinBackend() string {
	totalBackends := len(lb.backends)
	for i := 0; i < totalBackends; i++ {
		idx := atomic.AddUint64(&lb.counter, 1) % uint64(totalBackends)
		backend := lb.backends[idx]

		// Check if backend is healthy
		if lb.healthCheck[backend].Load() {
			return backend
		}
	}
	return ""
}

// LeastConnectionsBackend selects backend with fewest connections
func (lb *LoadBalancer) LeastConnectionsBackend() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	var selected string
	var minConnections int64 = 1<<63 - 1

	for backend, count := range lb.connCounts {
		if *count < minConnections {
			minConnections = *count
			selected = backend
		}
	}
	return selected
}

// IPHashBackend selects backend based on client IP
func (lb *LoadBalancer) IPHashBackend(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr // Fallback if error occurs
	}

	hasher := fnv.New32a()
	hasher.Write([]byte(ip))
	index := hasher.Sum32() % uint32(len(lb.backends))
	return lb.backends[index]
}

// ServeHTTP forwards requests to the selected backend
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.NextBackend(r)

	// Increment connection count
	atomic.AddInt64(lb.connCounts[backend], 1)
	defer atomic.AddInt64(lb.connCounts[backend], -1)

	ReverseProxy(backend).ServeHTTP(w, r)

}
