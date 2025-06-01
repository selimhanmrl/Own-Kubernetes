package proxy

import (
	"time"

	"github.com/selimhanmrl/Own-Kubernetes/client"
	"github.com/selimhanmrl/Own-Kubernetes/models"
)

type ServiceWatcher struct {
	client    *client.Client
	nodeProxy *NodeProxy
}

func (sw *ServiceWatcher) WatchServices() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		services, _ := sw.client.ListServices("")
		for _, service := range services {
			if service.Spec.Type == "NodePort" {
				sw.updateServiceRules(service)
			}
		}
	}
}

func (sw *ServiceWatcher) updateServiceRules(service models.Service) {
	// Update iptables rules for the service
	// Update proxy configuration
}

