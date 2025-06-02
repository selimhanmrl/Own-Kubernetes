package server

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/selimhanmrl/Own-Kubernetes/models"
)

type ProxyServer struct {
	services    map[string]*ServiceProxy
	mu          sync.RWMutex
	roundRobin  map[string]*uint32
	nodePortMap map[int]*ServiceProxy // Maps NodePort to service
}

type ServiceProxy struct {
	service  *models.Service
	backends []string // List of pod IPs with their ports
}

func NewProxyServer() *ProxyServer {
	return &ProxyServer{
		services:    make(map[string]*ServiceProxy),
		roundRobin:  make(map[string]*uint32),
		nodePortMap: make(map[int]*ServiceProxy),
	}
}

func (p *ProxyServer) RegisterService(service *models.Service, pods []models.Pod) {
	p.mu.Lock()
	defer p.mu.Unlock()

	backends := make([]string, 0)
	for _, pod := range pods {
		if pod.Status.Phase == "Running" {
			// Add pod's IP and port
			backend := fmt.Sprintf("http://%s:%d", pod.Status.HostIP, pod.Status.AssignedPort)
			backends = append(backends, backend)
		}
	}

	serviceProxy := &ServiceProxy{
		service:  service,
		backends: backends,
	}

	p.services[service.Metadata.Name] = serviceProxy
	p.roundRobin[service.Metadata.Name] = new(uint32)

	// Register NodePorts if service type is NodePort
	if service.Spec.Type == "NodePort" {
		for _, port := range service.Spec.Ports {
			if port.NodePort > 0 {
				p.nodePortMap[port.NodePort] = serviceProxy
			}
		}
	}
}

func (p *ProxyServer) Start() error {
	// Handle all NodePort services
	for nodePort, serviceProxy := range p.nodePortMap {
		go func(port int, proxy *ServiceProxy) {
			addr := fmt.Sprintf(":%d", port)
			fmt.Printf("🔄 Starting proxy for NodePort %d\n", port)

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				p.handleRequest(w, r, proxy)
			})

			if err := http.ListenAndServe(addr, handler); err != nil {
				fmt.Printf("❌ Failed to start proxy on port %d: %v\n", port, err)
			}
		}(nodePort, serviceProxy)
	}
	return nil
}

func (p *ProxyServer) handleRequest(w http.ResponseWriter, r *http.Request, proxy *ServiceProxy) {
	if len(proxy.backends) == 0 {
		http.Error(w, "No backends available", http.StatusServiceUnavailable)
		return
	}

	// Get the service name
	serviceName := proxy.service.Metadata.Name

	// Get the current counter for this service
	counter := p.roundRobin[serviceName]

	// Round-robin selection of backend
	index := int(atomic.AddUint32(counter, 1)-1) % len(proxy.backends)
	backend := proxy.backends[index]

	// Parse the backend URL
	targetURL, err := url.Parse(backend)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid backend URL: %v", err), http.StatusInternalServerError)
		return
	}

	// Create reverse proxy
	reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Update request host and headers
	r.URL.Host = targetURL.Host
	r.URL.Scheme = targetURL.Scheme
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))

	fmt.Printf("➡️ Proxying request to %s (backend %d of %d)\n",
		backend, index+1, len(proxy.backends))

	// Forward the request
	reverseProxy.ServeHTTP(w, r)
}

func (p *ProxyServer) RemoveService(serviceName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if proxy, exists := p.services[serviceName]; exists {
		// Remove NodePort mappings
		if proxy.service.Spec.Type == "NodePort" {
			for _, port := range proxy.service.Spec.Ports {
				delete(p.nodePortMap, port.NodePort)
			}
		}
		delete(p.services, serviceName)
		delete(p.roundRobin, serviceName)
	}
}
