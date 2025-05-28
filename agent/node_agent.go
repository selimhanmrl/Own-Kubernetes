package agent

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/selimhanmrl/Own-Kubernetes/client"
	"github.com/selimhanmrl/Own-Kubernetes/models"
)

func findServicesForPod(pod *models.Pod, client *client.Client) []models.Service {
	if pod.Metadata.Labels == nil {
		return nil
	}

	services, err := client.ListServices("")
	if err != nil {
		fmt.Printf("❌ Failed to list services: %v\n", err)
		return nil
	}

	var matchingServices []models.Service
	for _, svc := range services {
		if matchLabels(pod.Metadata.Labels, svc.Spec.Selector) {
			matchingServices = append(matchingServices, svc)
		}
	}

	return matchingServices
}

func matchLabels(podLabels, selector map[string]string) bool {
	if selector == nil {
		return false
	}

	for k, v := range selector {
		if podLabels[k] != v {
			return false
		}
	}
	return true
}

type NodeAgent struct {
	nodeName string
	nodeIP   string
	client   *client.Client
}

func NewNodeAgent(nodeName, nodeIP, apiHost, apiPort string) *NodeAgent {
	return &NodeAgent{
		nodeName: nodeName,
		nodeIP:   nodeIP,
		client: client.NewClient(client.ClientConfig{
			Host: apiHost,
			Port: apiPort,
		}),
	}
}

func (a *NodeAgent) Start() error {
	fmt.Printf("🚀 Starting node agent for node %s (IP: %s)\n", a.nodeName, a.nodeIP)
	fmt.Printf("📡 Connecting to API server at %s:%s\n", a.client.GetConfig().Host, a.client.GetConfig().Port)

	// Register node
	node := models.Node{
		Name: a.nodeName,
		IP:   a.nodeIP,
		Status: models.NodeStatus{
			Phase:         "Ready",
			LastHeartbeat: time.Now(),
			Conditions: []models.NodeCondition{
				{
					Type:           "Ready",
					Status:         "True",
					LastUpdateTime: time.Now(),
				},
			},
			Capacity: getNodeCapacity(),
		},
		Labels: make(map[string]string), // Initialize empty labels
	}

	// Try to register the node
	if err := a.client.RegisterNode(node); err != nil {
		return fmt.Errorf("❌ failed to register node: %v", err)
	}
	fmt.Printf("✅ Successfully registered node %s\n", a.nodeName)

	// Start monitoring pods
	fmt.Printf("👀 Starting pod monitor...\n")
	go a.monitorAndManagePods()

	// Start heartbeat
	fmt.Printf("💓 Starting heartbeat...\n")
	go a.startHeartbeat()

	return nil
}

func (a *NodeAgent) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		status := models.NodeStatus{
			Phase:         "Ready",
			LastHeartbeat: time.Now(),
			Conditions: []models.NodeCondition{
				{
					Type:           "Ready",
					Status:         "True",
					LastUpdateTime: time.Now(),
				},
			},
			Capacity: getNodeCapacity(),
		}

		if err := a.client.UpdateNodeStatus(a.nodeName, status); err != nil {
			fmt.Printf("Failed to update node status: %v\n", err)
		}
	}
}

func (a *NodeAgent) monitorAndManagePods() {
	ticker := time.NewTicker(10 * time.Second)
	previousPods := make(map[string]bool)
	for range ticker.C {
		fmt.Printf("🔍 Checking for pods assigned to node %s...\n", a.nodeName)
		currentPods := make(map[string]bool)

		pods, err := a.client.ListPods("")
		if err != nil {
			fmt.Printf("❌ Failed to list pods: %v\n", err)
			continue
		}

		// Process each pod
		for _, pod := range pods {
			currentPods[pod.Metadata.Name] = true

			fmt.Printf("📦 Found pod %s (status: %s, node: %s)\n",
				pod.Metadata.Name, pod.Status.Phase, pod.Spec.NodeName)

			if pod.Spec.NodeName == a.nodeName && pod.Status.Phase == "Pending" {
				fmt.Printf("🚀 Starting pod %s on node %s\n", pod.Metadata.Name, a.nodeName)

				// Start the pod's containers
				if err := a.startPod(&pod); err != nil {
					fmt.Printf("❌ Failed to start pod %s: %v\n", pod.Metadata.Name, err)
					continue
				}
				fmt.Printf("✅ Successfully started pod %s\n", pod.Metadata.Name)
			}
		}
		for podName := range previousPods {
			if !currentPods[podName] {
				fmt.Printf("🗑️ Pod %s was deleted, cleaning up containers\n", podName)
				if err := a.cleanupPod(podName); err != nil {
					fmt.Printf("❌ Failed to cleanup pod %s: %v\n", podName, err)
				}
			}
		}

		// Update previous pods map
		previousPods = currentPods

	}
}

func (a *NodeAgent) startPod(pod *models.Pod) error {
	// Create unique names for containers

	for _, container := range pod.Spec.Containers {
		containerName := pod.Metadata.Name

		// Get services that select this pod
		services := findServicesForPod(pod, a.client)

		// Check if container already exists and is running
		if isContainerRunning(containerName) {
			fmt.Printf("Container %s is already running\n", containerName)
			continue
		}

		// Convert resource limits
		memoryLimit := "512m" // default
		cpuLimit := "1"       // default
		if container.Resources.Limits != nil {
			if memory := container.Resources.Limits["memory"]; memory != "" {
				if converted, err := convertMemoryToDockerFormat(memory); err == nil {
					memoryLimit = converted
				}
			}
			if cpu := container.Resources.Limits["cpu"]; cpu != "" {
				if converted, err := convertCPU(cpu); err == nil {
					cpuLimit = converted
				}
			}
		}

		// Create container
		args := []string{
			"run", "-d",
			"--name", containerName,
			"--memory=" + memoryLimit,
			"--memory-swap=" + memoryLimit, // Disable swap
			"--cpus=" + cpuLimit,
			"--pids-limit=100",                 // Limit number of processes
			"--security-opt=no-new-privileges", // Restrict privileges
		}

		if len(services) > 0 {
			fmt.Printf("📦 Found %d services for pod %s\n", len(services), pod.Metadata.Name)
			for _, svc := range services {
				for _, port := range svc.Spec.Ports {
					if port.NodePort > 0 {
						portMapping := fmt.Sprintf("%d:%d", port.NodePort, port.TargetPort)
						args = append(args, "-p", portMapping)
					}
				}
			}
		}

		args = append(args, container.Image)

		cmd := exec.Command("docker", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			pod.Status.Phase = "Failed"
			a.client.UpdatePodStatus(pod)
			return fmt.Errorf("failed to start container: %v, output: %s", err, string(output))
		}

		// Get container ID
		containerId, err := getContainerId(containerName)
		if err != nil {
			fmt.Printf("Warning: Failed to get container ID: %v\n", err)
		}

		// Update pod status with container info
		pod.Status.ContainerID = containerId
		pod.Status.Phase = "Running"
		pod.Status.HostIP = a.nodeIP
		pod.Status.StartTime = time.Now().Format(time.RFC3339)

		fmt.Printf("✅ Started container %s with ID %s\n", containerName, containerId)
	}

	// Update final pod status
	return a.client.UpdatePodStatus(pod)
}

func isContainerRunning(containerName string) bool {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

func getNodeCapacity() models.ResourceList {
	// Get CPU info
	cmd := exec.Command("nproc")
	output, err := cmd.Output()
	cpuCount := "1"
	if err == nil {
		cpuCount = strings.TrimSpace(string(output))
	}

	// Get memory info
	cmd = exec.Command("free", "-m")
	output, err = cmd.Output()
	memoryMB := "1024"
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) > 1 {
				memoryMB = fields[1]
			}
		}
	}

	return models.ResourceList{
		"cpu":    cpuCount,
		"memory": fmt.Sprintf("%sMi", memoryMB),
	}
}

func (a *NodeAgent) getNodeResources() models.NodeResources {
	resources := models.NodeResources{
		CPU:    "4", // Default values
		Memory: "8Gi",
		Pods:   "110", // Maximum pods per node
	}

	// Get actual CPU cores
	if output, err := exec.Command("nproc").Output(); err == nil {
		resources.CPU = strings.TrimSpace(string(output))
	}

	// Get actual memory in GB
	if output, err := exec.Command("free", "-g").Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) > 1 {
				resources.Memory = fmt.Sprintf("%sGi", fields[1])
			}
		}
	}

	return resources
}

func convertMemoryToDockerFormat(memory string) (string, error) {
	memory = strings.ToLower(strings.TrimSpace(memory))
	var value float64
	var unit string

	if _, err := fmt.Sscanf(memory, "%f%s", &value, &unit); err != nil {
		return "", fmt.Errorf("invalid memory format: %s", memory)
	}

	switch unit {
	case "mi", "m":
		return fmt.Sprintf("%.0fm", value), nil
	case "gi", "g":
		return fmt.Sprintf("%.0fg", value), nil
	case "ki", "k":
		return fmt.Sprintf("%.0fk", value), nil
	default:
		return "", fmt.Errorf("unsupported memory unit: %s", unit)
	}
}

func convertCPU(cpu string) (string, error) {
	// Handle millicpu format (e.g., "250m")
	if strings.HasSuffix(cpu, "m") {
		value, err := strconv.ParseFloat(strings.TrimSuffix(cpu, "m"), 64)
		if err != nil {
			return "", fmt.Errorf("invalid CPU format: %s", cpu)
		}
		// Convert millicpu to CPU cores (e.g., 250m -> 0.25)
		return fmt.Sprintf("%.3f", value/1000.0), nil
	}

	// Handle whole CPU cores
	value, err := strconv.ParseFloat(cpu, 64)
	if err != nil {
		return "", fmt.Errorf("invalid CPU format: %s", cpu)
	}
	return fmt.Sprintf("%.3f", value), nil
}

func getContainerId(containerName string) (string, error) {
	cmd := exec.Command("docker", "inspect", "-f", "{{.Id}}", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (a *NodeAgent) cleanupPod(podName string) error {
	// Get container name
	containerName := podName

	// Check if container exists
	cmd := exec.Command("docker", "inspect", containerName)
	if err := cmd.Run(); err == nil {
		// Container exists, stop it first
		stopCmd := exec.Command("docker", "stop", containerName)
		if err := stopCmd.Run(); err != nil {
			return fmt.Errorf("failed to stop container %s: %v", containerName, err)
		}

		// Remove the container
		rmCmd := exec.Command("docker", "rm", containerName)
		if err := rmCmd.Run(); err != nil {
			return fmt.Errorf("failed to remove container %s: %v", containerName, err)
		}

		fmt.Printf("✅ Cleaned up container %s\n", containerName)
	}

	return nil
}
