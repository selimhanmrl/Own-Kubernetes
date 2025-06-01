package proxy

import (
	"fmt"
	"net"
	"io"
)


type NodeProxy struct {
    nodeIP     string
    services   map[string]*ServiceConfig
    listeners  map[int]net.Listener
}

func (np *NodeProxy) Start() error {
    for _, service := range np.services {
        // Create listener for each NodePort
        listener, err := net.Listen("tcp", fmt.Sprintf(":%d", service.NodePort))
        if err != nil {
            return err
        }
        
        np.listeners[service.NodePort] = listener
        
        // Start forwarding connections
        go np.handleConnections(listener, service)
    }
    return nil
}

func (np *NodeProxy) handleConnections(listener net.Listener, service *ServiceConfig) {
    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        
        go np.proxyConnection(conn, service)
    }
}

func (np *NodeProxy) proxyConnection(conn net.Conn, service *ServiceConfig) {
	defer conn.Close()
	
	// Choose a backend pod to forward the connection
	backend, err := np.chooseBackend(service)
	if err != nil {
		return
	}
	
	targetAddr := fmt.Sprintf("%s:%d", backend.PodIP, backend.Port)
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		return
	}
	defer targetConn.Close()
	
	// Forward data between the client and the backend pod
	go func() {
		io.Copy(targetConn, conn)
	}()
	io.Copy(conn, targetConn)
}

func (np *NodeProxy) chooseBackend(service *ServiceConfig) (*Backend, error) {
	// Simple round-robin selection of backend
	if len(service.Backends) == 0 {
		return nil, fmt.Errorf("no backends available for service %s", service.Name)
	}
	
	// For simplicity, just return the first backend
	// In a real implementation, you would implement round-robin or least connections logic
	return &service.Backends[0], nil
}