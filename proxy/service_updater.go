package proxy

import (
    "time"
    "log"
    "github.com/selimhanmrl/Own-Kubernetes/client"
    "github.com/selimhanmrl/Own-Kubernetes/models"
)

type ServiceUpdater struct {
    client       *client.Client
    loadBalancer *LoadBalancer
}

func NewServiceUpdater(client *client.Client, lb *LoadBalancer) *ServiceUpdater {
    return &ServiceUpdater{
        client:       client,
        loadBalancer: lb,
    }
}

func (su *ServiceUpdater) Start() {
    log.Println("Starting service updater...")
    go su.watchServices()
}

func (su *ServiceUpdater) watchServices() {
    ticker := time.NewTicker(10 * time.Second)
    for range ticker.C {
        services, err := su.client.ListServices("")
        if err != nil {
            log.Printf("Error fetching services: %v", err)
            continue
        }

        su.updateServices(services)
    }
}

func (su *ServiceUpdater) updateServices(services []models.Service) {
    su.loadBalancer.mutex.Lock()
    defer su.loadBalancer.mutex.Unlock()

    // Track current services for cleanup
    currentServices := make(map[string]bool)

    for _, service := range services {
        currentServices[service.Metadata.Name] = true

        if service.Spec.Type != "NodePort" {
            continue
        }

        config := &ServiceConfig{
            Name:       service.Metadata.Name,
            Namespace:  service.Metadata.Namespace,
            Protocol:   "TCP", // Default to TCP, can be made configurable
            Port:       service.Spec.Ports[0].Port,
            TargetPort: service.Spec.Ports[0].TargetPort,
            NodePort:   service.Spec.Ports[0].NodePort,
            Selector:   service.Spec.Selector,
            Backends:   make([]Backend, 0),
        }

        // Update backends for the service
        pods, err := su.client.ListPods(service.Metadata.Namespace)
        if err != nil {
            log.Printf("Error fetching pods for service %s: %v", service.Metadata.Name, err)
            continue
        }

        for _, pod := range pods {
            if matchLabels(pod.Metadata.Labels, service.Spec.Selector) {
                backend := Backend{
                    PodName: pod.Metadata.Name,
                    PodIP:   pod.Status.PodIP,
                    NodeIP:  pod.Status.HostIP,
                    Port:    service.Spec.Ports[0].TargetPort,
                }
                config.Backends = append(config.Backends, backend)
            }
        }

        su.loadBalancer.UpdateService(config)
    }

    // Cleanup removed services
    su.loadBalancer.CleanupServices(currentServices)
}

func matchLabels(podLabels, selector map[string]string) bool {
    for key, value := range selector {
        if podLabels[key] != value {
            return false
        }
    }
    return true
}