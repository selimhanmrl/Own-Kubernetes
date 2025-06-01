package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"


)

type ProxyServer struct {
	loadBalancer *LoadBalancer
	mutex        sync.RWMutex
}

func NewProxyServer(lb *LoadBalancer) *ProxyServer {
	return &ProxyServer{
		loadBalancer: lb,
	}
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serviceName := getServiceFromHost(r.Host)
	service := p.loadBalancer.GetService(serviceName)

	if service == nil {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	backend := p.loadBalancer.GetNextBackend(serviceName)
	if backend == nil {
		http.Error(w, "No backends available", http.StatusServiceUnavailable)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", backend.PodIP, service.TargetPort),
	})

	proxy.ServeHTTP(w, r)
}


func getServiceFromHost(host string) string {
	// Assuming the host is in the format "service-name.namespace.svc.cluster.local"
	// We can extract the service name from it
	parts := splitHost(host)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func splitHost(host string) []string {
	// Split the host by '.' and return the first part
	// This is a simple implementation; you might want to handle more complex cases
	return split(host, '.')
}

func split(s string, sep rune) []string {
	parts := []string{}
	current := ""
	for _, r := range s {
		if r == sep {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
