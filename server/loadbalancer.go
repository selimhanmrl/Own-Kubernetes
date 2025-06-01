package server

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/selimhanmrl/Own-Kubernetes/models"
)

type LoadBalancerHandler struct {
	services map[string]*ServiceConfig
	mutex    sync.RWMutex
}

type ServiceConfig struct {
	Service  models.Service
	Backends []string // Pod IPs
}

func NewLoadBalancerHandler() *LoadBalancerHandler {
	return &LoadBalancerHandler{
		services: make(map[string]*ServiceConfig),
	}
}

func (h *LoadBalancerHandler) RegisterService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var service models.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.mutex.Lock()
	h.services[service.Metadata.Name] = &ServiceConfig{
		Service:  service,
		Backends: make([]string, 0),
	}
	h.mutex.Unlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Service registered successfully",
	})
}
