package main

import (
	"groxy/middleware"
	"groxy/proxy"
	"groxy/utils"
	"log"
	"net/http"
	"strconv"
)

func main() {
	// Load Config
	config := utils.LoadConfig()

	// Initialize Load Balancer
	lb := proxy.NewLoadBalancer(config.LoadBalancer.Backends, config.LoadBalancer.Strategy)

	// Define HTTP Server
	mux := http.NewServeMux()
	mux.Handle("/", middleware.LoggingMiddleware(middleware.RateLimitMiddleware(lb)))

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(config.Server.Port),
		Handler: mux,
	}

	// Start Server
	log.Println("Starting proxy server on port", config.Server.Port)
	if config.Server.TLS.Enabled {
		log.Fatal(server.ListenAndServeTLS(config.Server.TLS.CertFile, config.Server.TLS.KeyFile))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}
