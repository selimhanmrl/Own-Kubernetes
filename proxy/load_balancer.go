package proxy

import (
	"sync"
)

type LoadBalancer struct {
	services   map[string]*ServiceConfig
	mutex      sync.RWMutex
	portMapper *PortMapper
}

type ServiceConfig struct {
	Name       string
	Namespace  string
	Protocol   string
	Port       int
	TargetPort int
	NodePort   int
	Selector   map[string]string
	Backends   []Backend
}

type Backend struct {
	PodName string
	PodIP   string
	NodeIP  string
	Port    int
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		services:   make(map[string]*ServiceConfig),
		portMapper: NewPortMapper(),
	}
}

func (lb *LoadBalancer) AddService(service *ServiceConfig) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Assign a NodePort if not specified
	if service.NodePort == 0 {
		service.NodePort = lb.portMapper.AssignPort()
	}

	lb.services[service.Name] = service
}

func (lb *LoadBalancer) GetService(name string) *ServiceConfig {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	service, exists := lb.services[name]
	if !exists {
		return nil
	}
	return service
}

func (lb *LoadBalancer) GetNextBackend(serviceName string) *Backend {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	service, exists := lb.services[serviceName]
	if !exists || len(service.Backends) == 0 {
		return nil
	}

	// Round-robin selection of backend
	backend := &service.Backends[0]
	service.Backends = append(service.Backends[1:], *backend) // Rotate the slice
	return backend
}

func (lb *LoadBalancer) UpdateService(config *ServiceConfig) {
	lb.services[config.Name] = config
}

func (lb *LoadBalancer) CleanupServices(currentServices map[string]bool) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	for name := range lb.services {
		if !currentServices[name] {
			delete(lb.services, name)
		}
	}
}

func (lb *LoadBalancer) RegisterService(service *ServiceConfig) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	// Assign a NodePort if not specified
	if service.NodePort == 0 {
		service.NodePort = lb.portMapper.AssignPort()
	}

	lb.services[service.Name] = service
}
