package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"github.com/selimhanmrl/Own-Kubernetes/models"
)

type ClientConfig struct {
	Host string
	Port string
}

type Client struct {
	baseURL      string
	config       ClientConfig
	assignedPods map[string]int // Track assigned NodePorts for pods
}

func NewClient(config ClientConfig) *Client {
	if config.Host == "" {
		config.Host = "localhost"
	}
	if config.Port == "" {
		config.Port = "8080"
	}

	c := &Client{
		baseURL: fmt.Sprintf("http://%s:%s", config.Host, config.Port),
		config:  config,
	}

	// Load existing node port assignments
	if err := c.loadNodePorts(); err != nil {
		fmt.Printf("Warning: Could not load node port assignments: %v\n", err)
		c.assignedPods = make(map[string]int)
	}

	return c
}

func (c *Client) ListPods(namespace string) ([]models.Pod, error) {
	url := c.baseURL + "/api/v1/pods"
	if namespace != "" {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/pods", c.baseURL, namespace)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list pods: %s", resp.Status)
	}

	var pods []models.Pod
	if err := json.NewDecoder(resp.Body).Decode(&pods); err != nil {
		return nil, err
	}
	return pods, nil
}

func (c *Client) CreatePod(pod models.Pod) error {
	url := c.baseURL + "/api/v1/pods"
	data, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create pod: %s", resp.Status)
	}
	return nil
}

// Add GetNode method if not already present
func (c *Client) GetNode(name string) (*models.Node, error) {
	url := fmt.Sprintf("%s/api/v1/nodes/%s", c.baseURL, name)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get node: %s", resp.Status)
	}

	var node models.Node
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, fmt.Errorf("failed to decode node: %v", err)
	}

	return &node, nil
}

func (c *Client) DeletePod(namespace, name string) error {
	// First try to get the pod
	pod, err := c.GetPod(name)
	if err != nil {
		if strings.Contains(err.Error(), "404 Not Found") {
			fmt.Printf("⚠️ Pod '%s' not found in API server, checking nodes directly...\n", name)
			// Try to cleanup from nodes even if pod is not in API server
			return c.cleanupPodFromNodes(name)
		}
		return fmt.Errorf("failed to get pod info: %v", err)
	}

	// If pod exists and is assigned to a node, cleanup from node first
	if pod.Spec.NodeName != "" {
		node, err := c.GetNode(pod.Spec.NodeName)
		if err != nil {
			fmt.Printf("⚠️ Warning: Failed to get node info: %v\n", err)
		} else {
			fmt.Printf("🗑️ Cleaning up pod '%s' from node '%s'\n", name, node.Name)
			if err := c.cleanupPodFromNode(name, node); err != nil {
				fmt.Printf("⚠️ Warning: Failed to cleanup from node: %v\n", err)
			}
		}
	}

	// Delete from API server
	url := fmt.Sprintf("%s/api/v1/pods/%s", c.baseURL, name)
	fmt.Printf("🗑️ Deleting pod from API server: %s\n", url)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("ℹ️ Pod '%s' already deleted from API server\n", name)
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete pod: %s - %s", resp.Status, string(body))
	}

	fmt.Printf("✅ Successfully deleted pod '%s'\n", name)
	return nil
}

// Add these helper functions
func (c *Client) cleanupPodFromNodes(podName string) error {
	// Get all nodes
	nodes, err := c.ListNodes()
	if err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}

	var lastErr error
	cleaned := false
	for _, node := range nodes {
		fmt.Printf("🔍 Checking node '%s' for pod '%s'\n", node.Name, podName)
		if err := c.cleanupPodFromNode(podName, &node); err != nil {
			lastErr = err
			fmt.Printf("⚠️ Failed to cleanup from node %s: %v\n", node.Name, err)
		} else {
			cleaned = true
		}
	}

	if !cleaned && lastErr != nil {
		return fmt.Errorf("failed to cleanup pod from any node: %v", lastErr)
	}
	return nil
}

func (c *Client) cleanupPodFromNode(podName string, node *models.Node) error {
	nodeURL := fmt.Sprintf("http://%s:8081/pods/%s", node.IP, podName)
	req, err := http.NewRequest(http.MethodDelete, nodeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create node request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact node server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("node server failed to delete pod: %s - %s", resp.Status, string(body))
	}

	fmt.Printf("✅ Successfully cleaned up pod '%s' from node '%s'\n", podName, node.Name)
	return nil
}

// Add helper method to get pod details
func (c *Client) GetPod(name string) (*models.Pod, error) {
	url := fmt.Sprintf("%s/api/v1/pods/%s", c.baseURL, name)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get pod: %s", resp.Status)
	}

	var pod models.Pod
	if err := json.NewDecoder(resp.Body).Decode(&pod); err != nil {
		return nil, fmt.Errorf("failed to decode pod: %v", err)
	}

	return &pod, nil
}

func (c *Client) UpdatePod(pod models.Pod) error {
	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s", c.baseURL, pod.Metadata.Namespace, pod.Metadata.Name)
	data, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update pod: %s", resp.Status)
	}
	return nil
}

func (c *Client) CreateService(service models.Service) error {
	if service.Spec.Type == "NodePort" {
		fmt.Println("📡 Processing NodePort service...")

		// Get matching pods
		pods, err := c.ListPods("")
		if err != nil {
			return fmt.Errorf("failed to list pods: %v", err)
		}

		// Track used ports to ensure uniqueness
		usedPorts := make(map[int]bool)
		for _, port := range c.assignedPods {
			usedPorts[port] = true
		}

		// First matching pod's port will be used for the service
		var serviceNodePort int

		// Find matching pods and assign unique NodePorts if needed
		for _, pod := range pods {
			if matchLabels(pod.Metadata.Labels, service.Spec.Selector) {
				// Check if pod already has a NodePort
				if existingPort, exists := c.assignedPods[pod.Metadata.Name]; exists {
					if serviceNodePort == 0 {
						serviceNodePort = existingPort
					}
					fmt.Printf("🔗 Pod '%s' already has NodePort %d\n", pod.Metadata.Name, existingPort)
					continue
				}

				// Generate new unique NodePort
				var nodePort int
				for {
					nodePort = generateNodePort()
					if !usedPorts[nodePort] {
						break
					}
				}

				// Assign the new port
				usedPorts[nodePort] = true
				c.assignedPods[pod.Metadata.Name] = nodePort
				fmt.Printf("🔗 Pod '%s' assigned new NodePort %d\n", pod.Metadata.Name, nodePort)

				if serviceNodePort == 0 {
					serviceNodePort = nodePort
				}
			}
		}

		// Set the NodePort for the service (using the first pod's port)
		if serviceNodePort > 0 {
			fmt.Printf("🔌 Assigned NodePort %d for port %d\n", serviceNodePort, service.Spec.Ports[0].Port)
			service.Spec.Ports[0].NodePort = serviceNodePort
		}

		// Add service to load balancer
		if err := c.registerServiceWithLoadBalancer(service); err != nil {
			return fmt.Errorf("failed to register with load balancer: %v", err)
		}
	}

	// Create the service
	fmt.Printf("📦 Creating service '%s'...\n", service.Metadata.Name)
	url := fmt.Sprintf("%s/api/v1/services", c.baseURL)
	data, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create service: %s", resp.Status)
	}

	fmt.Printf("✅ Service '%s' created successfully\n", service.Metadata.Name)

	// Show final pod-service mappings
	pods, _ := c.ListPods("")
	for _, pod := range pods {
		if matchLabels(pod.Metadata.Labels, service.Spec.Selector) {
			if port, exists := c.assignedPods[pod.Metadata.Name]; exists {
				fmt.Printf("🔗 Pod '%s' matches service selector %d\n", pod.Metadata.Name, port)
			}
		}
	}

	// After assigning ports to pods
	if err := c.saveNodePorts(); err != nil {
		fmt.Printf("Warning: Could not save node port assignments: %v\n", err)
	}

	return nil
}

func (c *Client) ListServices(namespace string) ([]models.Service, error) {
	url := fmt.Sprintf("%s/api/v1/services", c.baseURL)
	if namespace != "" {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/services", c.baseURL, namespace)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get services: %s", resp.Status)
	}

	var services []models.Service
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return nil, fmt.Errorf("failed to decode services: %v", err)
	}

	return services, nil
}

func (c *Client) RegisterNode(node models.Node) error {
	url := fmt.Sprintf("%s/api/v1/nodes", c.baseURL)

	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to register node: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to register node: %s", resp.Status)
	}

	return nil
}

func (c *Client) UpdateNodeStatus(nodeName string, status models.NodeStatus) error {
	url := fmt.Sprintf("%s/api/v1/nodes/%s/status", c.baseURL, nodeName)
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update node status: %s", resp.Status)
	}
	return nil
}

func (c *Client) ListNodes() ([]models.Node, error) {
	// Fix the URL path to match the API server routes
	url := c.baseURL + "/api/v1/nodes"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list nodes: %s", resp.Status)
	}

	var nodes []models.Node
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	return nodes, nil
}

func (c *Client) GetConfig() ClientConfig {
	return c.config
}

// Add this method after other client methods
func (c *Client) UpdatePodStatus(pod *models.Pod) error {
	url := fmt.Sprintf("%s/api/v1/pods/%s/status", c.baseURL, pod.Metadata.Name)
	fmt.Printf("🔄 Updating pod status: %s\n", url)

	data, err := json.Marshal(pod)
	if err != nil {
		return fmt.Errorf("failed to marshal pod: %v", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body for error details
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("✅ Successfully updated pod status for %s\n", pod.Metadata.Name)
	return nil
}

// Helper function to generate a random NodePort in the range 30000-32767
func generateNodePort() int {
	return 30000 + rand.Intn(2768) // Range: 30000-32767
}

func matchLabels(podLabels, selector map[string]string) bool {
	for key, value := range selector {
		if podLabels[key] != value {
			return false
		}
	}
	return true
}

func (c *Client) GetAssignedNodePort(podName string) (int, bool) {
	port, exists := c.assignedPods[podName]
	return port, exists
}

// Add method to save node port assignments to file
func (c *Client) saveNodePorts() error {
	data, err := json.Marshal(c.assignedPods)
	if err != nil {
		return err
	}
	return os.WriteFile("nodeports.json", data, 0644)
}

// Add method to load node port assignments from file
func (c *Client) loadNodePorts() error {
	data, err := os.ReadFile("nodeports.json")
	if err != nil {
		if os.IsNotExist(err) {
			c.assignedPods = make(map[string]int)
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &c.assignedPods)
}

func (c *Client) registerServiceWithLoadBalancer(service models.Service) error {
	url := fmt.Sprintf("%s/api/v1/loadbalancer/services", c.baseURL)
	data, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service for load balancer: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to register service with load balancer: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("load balancer registration failed: %s - %s", resp.Status, string(body))
	}

	fmt.Printf("✅ Service '%s' registered with load balancer successfully\n", service.Metadata.Name)
	return nil
}
