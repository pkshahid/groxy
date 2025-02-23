package proxy

import (
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

type LoadBalancer struct {
	backends    []string
	counter     uint64
	healthCheck map[string]*atomic.Bool
}

// NewLoadBalancer initializes a load balancer with backends
func NewLoadBalancer(backends []string) *LoadBalancer {
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
func (lb *LoadBalancer) NextBackend() string {
	totalBackends := len(lb.backends)
	if totalBackends == 0 {
		log.Println("No backends available")
		return "" // or handle error gracefully
	}
	for i := 0; i < totalBackends; i++ {
		idx := atomic.AddUint64(&lb.counter, 1) % uint64(totalBackends)
		backend := lb.backends[idx]

		// Check if backend is healthy
		if lb.healthCheck[backend].Load() {
			return backend
		}
	}
	return "" // No healthy backends available
}

// ServeHTTP forwards requests to the selected backend
func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for attempt := 0; attempt < len(lb.backends); attempt++ {
		backend := lb.NextBackend()
		if backend == "" {
			http.Error(w, "No healthy backends available", http.StatusServiceUnavailable)
			return
		}

		proxyReq, err := http.NewRequest(r.Method, backend+r.RequestURI, r.Body)
		if err != nil {
			log.Println("Error creating proxy request:", err)
			continue
		}
		proxyReq.Header = r.Header

		client := &http.Client{Timeout: 3 * time.Second} // Backend timeout
		resp, err := client.Do(proxyReq)
		if err != nil {
			log.Printf("[Failover] Backend %s failed, trying next...\n", backend)
			lb.healthCheck[backend].Store(false) // Mark backend as down immediately
			continue                             // Try next backend
		}

		// Forward response
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		resp.Body.Close()
		return
	}

	// No backend succeeded
	http.Error(w, "All backends failed", http.StatusServiceUnavailable)
}
