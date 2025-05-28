package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/selimhanmrl/Own-Kubernetes/models"
)

type ClientConfig struct {
	Host string
	Port string
}

type Client struct {
	baseURL string
	config  ClientConfig
}

func NewClient(config ClientConfig) *Client {
	if config.Host == "" {
		config.Host = "localhost"
	}
	if config.Port == "" {
		config.Port = "8080"
	}
	return &Client{
		baseURL: fmt.Sprintf("http://%s:%s", config.Host, config.Port),
		config:  config,
	}
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

func (c *Client) DeletePod(namespace, name string) error {
	// Use the same URL format as our other endpoints
	url := fmt.Sprintf("%s/api/v1/pods/%s", c.baseURL, name)

	fmt.Printf("🗑️ Deleting pod at: %s\n", url)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body for better error messages
	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete pod: %s - %s", resp.Status, string(body))
	}

	return nil
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
	url := c.baseURL + "/api/v1/services"
	data, err := json.Marshal(service)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create service: %s", resp.Status)
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
	url := c.baseURL + "/api/v1/nodes"
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
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