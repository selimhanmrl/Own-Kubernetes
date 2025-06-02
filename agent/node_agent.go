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

func findServicesForPod(pod *models.Pod, client *client.Client) ([]models.Service, error) {
	if pod.Metadata.Labels == nil {
		return nil, nil
	}

	services, err := client.ListServices(pod.Metadata.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %v", err)
	}

	var matchingServices []models.Service
	for _, svc := range services {
		if matchLabels(pod.Metadata.Labels, svc.Spec.Selector) {
			matchingServices = append(matchingServices, svc)
		}
	}

	return matchingServices, nil
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
				if err := a.StartPod(&pod); err != nil {
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

func (a *NodeAgent) StartPod(pod *models.Pod) error {
	fmt.Printf("🔄 Starting pod %s (initial AssignedPort: %d)\n",
		pod.Metadata.Name, pod.Status.AssignedPort)
	pod.Status.HostIP = a.nodeIP
	fmt.Printf("📍 Setting pod HostIP to: %s\n", a.nodeIP)

	if err := a.client.UpdatePodStatus(pod); err != nil {
		fmt.Printf("⚠️ Failed to update pod's HostIP: %v\n", err)
	}
	// Get fresh pod data with retries

	updatedPod, _ := a.client.GetPod(pod.Metadata.Name)
	if updatedPod.Status.AssignedPort > 0 {
		pod = updatedPod
	}

	// Get matching services first
	services, err := findServicesForPod(pod, a.client)
	fmt.Printf("🔍 Searching for services matching pod %s...\n", pod.Metadata.Name)
	fmt.Printf("🔍 Found %d services for pod %s\n", len(services), pod.Metadata.Name)
	fmt.Printf("🔍 Services: %v\n", services)

	if err != nil {
		fmt.Printf("⚠️ Warning: Failed to get services: %v\n", err)
	}

	fmt.Printf("🔍 Found %d matching services for pod\n", len(services))

	for _, container := range pod.Spec.Containers {
		containerName := pod.Metadata.Name

		if isContainerRunning(containerName) {
			fmt.Printf("⚠️ Container %s already exists, skipping...\n", containerName)
			continue
		}
		// Check if container already exists
		cmd := exec.Command("docker", "inspect", containerName)
		if err := cmd.Run(); err == nil {
			fmt.Printf("⚠️ Container %s already exists, skipping...\n", containerName)
			continue
		}

		// Defaults
		memoryLimit := "512m"
		cpuLimit := "1.0"

		if container.Resources.Limits != nil {
			if memory := container.Resources.Limits["memory"]; memory != "" {
				if converted, err := convertMemoryToDockerFormat(memory); err == nil {
					memoryLimit = converted
					fmt.Printf("📦 Using memory limit: %s\n", memoryLimit)
				} else {
					fmt.Printf("⚠️ Memory conversion failed: %v\n", err)
				}
			}
			if cpu := container.Resources.Limits["cpu"]; cpu != "" {
				if converted, err := convertCPU(cpu); err == nil {
					cpuLimit = converted
					fmt.Printf("⚙️  Using CPU limit: %s\n", cpuLimit)
				} else {
					fmt.Printf("⚠️ CPU conversion failed: %v\n", err)
				}
			}
		}

		args := []string{
			"run", "-d",
			"--name", containerName,
			"--memory=" + memoryLimit,
			"--cpus=" + cpuLimit,
		}

		// Track used ports to avoid conflicts
		//usedPorts := make(map[int]bool)

		// Add NodePort mappings first
		for _, svc := range services {
			if svc.Spec.Type == "NodePort" {
				for _, port := range svc.Spec.Ports {
					if pod.Status.AssignedPort > 0 && port.NodePort > 0 {
						portMapping := fmt.Sprintf("%d:%d", pod.Status.AssignedPort, port.TargetPort)
						args = append(args, "-p", portMapping)
						fmt.Printf("🔌 Mapping assigned port %d -> target port %d\n",
							pod.Status.AssignedPort, port.TargetPort)
					}
				}
			}
		}
		args = append(args, container.Image)

		fmt.Printf("🔧 Starting container with args: docker %s\n", strings.Join(args, " "))
		cmd = exec.Command("docker", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("❌ Failed to start container: %v\nOutput: %s", err, string(output))
		}

		fmt.Printf("✅ Started container %s\n", containerName)
	}

	pod.Status.Phase = "Running"
	return a.client.UpdatePodStatus(pod)
}

func isContainerRunning(name string) bool {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", name)
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

func (a *NodeAgent) ListPods() ([]models.Pod, error) {
	// List pods assigned to this node
	return a.client.ListPods(fmt.Sprintf("?fieldSelector=spec.nodeName=%s", a.nodeName))
}

func (a *NodeAgent) CleanupPod(podName string) error {
	// Remove all containers for this pod
	cmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=%s", podName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list containers: %v", err)
	}

	containerIDs := strings.Split(string(output), "\n")
	for _, id := range containerIDs {
		if id == "" {
			continue
		}
		cmd := exec.Command("docker", "rm", "-f", id)
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ Failed to remove container %s: %v\n", id, err)
		}
	}
	return nil
}

// Add this method to NodeAgent struct
func (a *NodeAgent) GetClient() *client.Client {
	return a.client
}
