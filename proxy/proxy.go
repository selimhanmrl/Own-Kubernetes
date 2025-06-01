package proxy

import (
	"sync"
	"fmt"
	"net"
	"time"

	"github.com/selimhanmrl/Own-Kubernetes/client"
)

type Proxy struct {
	services map[string]*ServiceConfig
	iptables *IPTables
	mutex    sync.RWMutex
	client   *Client // Assuming Client is defined elsewhere for service discovery
}

// type ServiceConfig struct {
// 	Name        string
// 	ClusterIP   string
// 	Protocol    string
// 	Port        int
// 	TargetPorts map[string]int // pod IP -> port
// 	NodePorts   map[string]int // node IP -> port
// }

type IPTables struct {
	rules []Rule
}

type Rule struct {
	Chain    string
	Source   string
	Target   string
	Protocol string
	Port     int
}


func NewProxy() *Proxy {
	return &Proxy{
		services: make(map[string]*ServiceConfig),
		iptables: &IPTables{},
	}
}

func (p *Proxy) RegisterService(service *ServiceConfig) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if service already exists
	if _, exists := p.services[service.Name]; exists {
		return // Service already registered
	}

	// Register the service
	p.services[service.Name] = service

	// Update iptables rules
	if err := p.updateIPTableRules(service); err != nil {
		// Handle error (e.g., log it)
	}
}

func (p *Proxy) updateEndpoints(endpoints map[string][]string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for serviceName, podIPs := range endpoints {
		if service, exists := p.services[serviceName]; exists {
			service.TargetPorts = make(map[string]int)
			for _, podIP := range podIPs {
				// Assuming a default port for simplicity
				service.TargetPorts[podIP] = 80 // Default port, can be customized
			}
		}
	}
}

// Add iptables/network rule management
func (p *Proxy) updateIPTableRules(service *ServiceConfig) error {
    rule := Rule{
        Chain:    "KUBE-SERVICES",
        Target:   service.ClusterIP,
        Protocol: service.Protocol,
        Port:    service.Port,
    }
    return p.iptables.AddRule(rule)
}

// Add service discovery
func (p *Proxy) watchEndpoints() {
    for {
        endpoints := p.client.GetEndpoints()
        p.updateEndpoints(endpoints)
        time.Sleep(10 * time.Second)
    }
}

// Add connection forwarding
func (p *Proxy) forwardConnection(conn net.Conn, service *ServiceConfig) {
    backend := p.chooseBackend(service)
    target := fmt.Sprintf("%s:%d", backend.IP, backend.Port)
    proxy := NewTCPProxy(conn, target)
    proxy.Start()
}

func (iptables *IPTables) AddRule(rule Rule) error {
	iptables.rules = append(iptables.rules, rule)
	// Here you would typically execute a command to add the rule to the system's iptables
	// For example: exec.Command("iptables", "-A", rule.Chain, "-p", rule.Protocol, "--dport", fmt.Sprint(rule.Port), "-j", rule.Target).Run()
	return nil
}

