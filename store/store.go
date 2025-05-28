package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"

	"github.com/go-redis/redis"
	"github.com/selimhanmrl/Own-Kubernetes/models"
	own_redis "github.com/selimhanmrl/Own-Kubernetes/redis"
)

var (
	mu        sync.Mutex
	nodeStore = []models.Node{
		{Name: "node1", IP: "192.168.1.10"},
	}
)

func SavePod(pod models.Pod) error {
	if own_redis.RedisClient == nil {
		return fmt.Errorf("RedisClient is not initialized")
	}

	if pod.Metadata.Namespace == "" {
		pod.Metadata.Namespace = "default"
	}

	// Use consistent key format: pods:{namespace}:{name}
	key := fmt.Sprintf("pods:%s:%s", pod.Metadata.Namespace, pod.Metadata.Name)

	fmt.Printf("💾 Saving pod to Redis with key: %s\n", key)

	value, err := json.Marshal(pod)
	if err != nil {
		return fmt.Errorf("failed to marshal pod: %v", err)
	}

	err = own_redis.RedisClient.Set(own_redis.Ctx, key, value, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to save pod: %v", err)
	}

	fmt.Printf("✅ Pod '%s' saved to Redis in namespace '%s'\n",
		pod.Metadata.Name, pod.Metadata.Namespace)
	return nil
}

func SaveReplicaSet(rs models.ReplicaSet) error {
	if own_redis.RedisClient == nil { // Use RedisClient from the redis package
		log.Fatalf("❌ RedisClient is not initialized")
	}
	if rs.Metadata.Namespace == "" {
		rs.Metadata.Namespace = "default" // Default to 'default' namespace
	}

	key := fmt.Sprintf("replicaset:%s:%s", rs.Metadata.Namespace, rs.Metadata.Name)

	// Convert ReplicaSet to JSON
	data, err := json.Marshal(rs)
	if err != nil {
		return fmt.Errorf("failed to marshal ReplicaSet: %v", err)
	}

	// Save to Redis
	err = own_redis.RedisClient.Set(own_redis.Ctx, key, data, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to save ReplicaSet to Redis: %v", err)
	}

	fmt.Printf("✅ ReplicaSet '%s' saved to Redis in namespace '%s'\n",
		rs.Metadata.Name, rs.Metadata.Namespace)
	return nil
}

func SaveService(service models.Service) error {
	if service.Spec.Type == "NodePort" {
		// Auto-assign NodePort if not specified
		for i := range service.Spec.Ports {
			if service.Spec.Ports[i].NodePort == 0 {
				service.Spec.Ports[i].NodePort = generateNodePort()
			}
		}
	}

	if own_redis.RedisClient == nil { // Use RedisClient from the redis package
		return fmt.Errorf("❌ RedisClient is not initialized")
	}

	if service.Metadata.Namespace == "" {
		service.Metadata.Namespace = "default" // Default to 'default' namespace
	}

	key := fmt.Sprintf("services:%s:%s", service.Metadata.Namespace, service.Metadata.Name) // Include namespace in the key
	value, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service: %v", err)
	}

	err = own_redis.RedisClient.Set(own_redis.Ctx, key, value, 0).Err()
	if err != nil {
		return fmt.Errorf("❌ Failed to save service '%s': %v", service.Metadata.Name, err)
	}

	fmt.Printf("✅ Service '%s' saved to Redis in namespace '%s'.\n", service.Metadata.Name, service.Metadata.Namespace)
	return nil
}

func GetPod(name string) (models.Pod, bool) {
	if name == "" {
		return models.Pod{}, false
	}

	// Use consistent key format
	key := fmt.Sprintf("pods:default:%s", name)
	fmt.Printf("🔍 Looking up pod with key: %s\n", key)

	value, err := own_redis.RedisClient.Get(own_redis.Ctx, key).Result()
	if err == redis.Nil {
		fmt.Printf("❌ Pod '%s' not found\n", name)
		return models.Pod{}, false
	} else if err != nil {
		fmt.Printf("❌ Error getting pod '%s': %v\n", name, err)
		return models.Pod{}, false
	}

	var pod models.Pod
	if err := json.Unmarshal([]byte(value), &pod); err != nil {
		fmt.Printf("❌ Error unmarshaling pod '%s': %v\n", name, err)
		return models.Pod{}, false
	}

	fmt.Printf("✅ Found pod '%s'\n", name)
	return pod, true
}

func DeletePodByName(name string, namespace string) bool {
	mu.Lock()
	defer mu.Unlock()

	if namespace == "" {
		namespace = "default" // Default to 'default' namespace
	}

	// List all pods in the specified namespace
	pods := ListPods(namespace)

	// Find the pod by name
	var uid string
	var pod models.Pod
	found := false
	for _, p := range pods {
		if p.Metadata.Name == name {
			uid = p.Metadata.UID
			pod = p
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("❌ Pod with name '%s' not found in namespace '%s'.\n", name, namespace)
		return false
	}

	// Stop the Docker container using the ContainerID
	if pod.Status.Phase == "Running" && pod.Status.ContainerID != "" {
		fmt.Printf("Stopping container with ID '%s'...\n", pod.Status.ContainerID)
		err := exec.Command("docker", "stop", pod.Status.ContainerID).Run()
		if err != nil {
			fmt.Printf("❌ Failed to stop container '%s': %v\n", pod.Status.ContainerID, err)
		} else {
			fmt.Printf("✅ Stopped container with ID '%s'.\n", pod.Status.ContainerID)
		}
	}

	// Delete the pod from Redis
	key := fmt.Sprintf("pods:%s:%s", namespace, uid)
	err := own_redis.RedisClient.Del(own_redis.Ctx, key).Err()
	if err != nil {
		fmt.Printf("❌ Failed to delete pod '%s' in namespace '%s': %v\n", name, namespace, err)
		return false
	}

	fmt.Printf("✅ Pod '%s' deleted successfully from namespace '%s'.\n", name, namespace)
	return true
}

func ListAllPods() []models.Pod {
	pattern := "pods:*" // Match all pods across all namespaces
	keys, err := own_redis.RedisClient.Keys(own_redis.Ctx, pattern).Result()
	if err != nil {
		fmt.Printf("❌ Failed to list pods: %v\n", err)
		return nil
	}

	var pods []models.Pod
	for _, key := range keys {
		value, _ := own_redis.RedisClient.Get(own_redis.Ctx, key).Result()
		var pod models.Pod
		json.Unmarshal([]byte(value), &pod)
		pods = append(pods, pod)
	}
	return pods
}

func ListPods(namespace string) []models.Pod {
	if namespace == "" {
		namespace = "default"
	}

	// Use consistent key pattern
	pattern := fmt.Sprintf("pods:%s:*", namespace)
	fmt.Printf("🔍 Listing pods with pattern: %s\n", pattern)

	keys, err := own_redis.RedisClient.Keys(own_redis.Ctx, pattern).Result()
	if err != nil {
		fmt.Printf("❌ Failed to list pods: %v\n", err)
		return nil
	}

	var pods []models.Pod
	for _, key := range keys {
		value, err := own_redis.RedisClient.Get(own_redis.Ctx, key).Result()
		if err != nil {
			fmt.Printf("❌ Error getting pod for key '%s': %v\n", key, err)
			continue
		}

		var pod models.Pod
		if err := json.Unmarshal([]byte(value), &pod); err != nil {
			fmt.Printf("❌ Error unmarshaling pod for key '%s': %v\n", key, err)
			continue
		}
		pods = append(pods, pod)
	}

	fmt.Printf("✅ Found %d pods in namespace '%s'\n", len(pods), namespace)
	return pods
}

func DeletePod(namespace, name string) error {
	if namespace == "" {
		namespace = "default"
	}

	// Get pod before deleting to get container info
	pod, found := GetPod(name)
	if !found {
		return fmt.Errorf("pod '%s' not found in namespace '%s'", name, namespace)
	}

	// Get the worker node where the pod is running
	nodeName := pod.Spec.NodeName
	if nodeName != "" {
		// Publish deletion event for the node agent
		event := map[string]string{
			"type":     "delete",
			"podName":  name,
			"nodeName": nodeName,
		}
		eventData, _ := json.Marshal(event)
		own_redis.RedisClient.Publish(own_redis.Ctx, "pod:events", string(eventData))
	}

	// Delete from Redis
	key := fmt.Sprintf("pods:%s:%s", namespace, name)
	err := own_redis.RedisClient.Del(own_redis.Ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete pod: %v", err)
	}

	fmt.Printf("✅ Pod '%s' deleted from store\n", name)
	return nil
}
func AddNode(node models.Node) {
	mu.Lock()
	defer mu.Unlock()
	nodeStore = append(nodeStore, node) // Append the new node to the slice
}

func SaveNode(node models.Node) error {
	if own_redis.RedisClient == nil {
		return fmt.Errorf("RedisClient is not initialized")
	}

	key := fmt.Sprintf("nodes:%s", node.Name)
	value, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal node: %v", err)
	}

	err = own_redis.RedisClient.Set(own_redis.Ctx, key, value, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to save node '%s': %v", node.Name, err)
	}

	fmt.Printf("✅ Node '%s' saved to Redis\n", node.Name)
	return nil
}

func UpdateNodeStatus(nodeName string, status models.NodeStatus) error {
	if own_redis.RedisClient == nil {
		return fmt.Errorf("RedisClient is not initialized")
	}

	key := fmt.Sprintf("nodes:%s", nodeName)
	value, err := own_redis.RedisClient.Get(own_redis.Ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to get node '%s': %v", nodeName, err)
	}

	var node models.Node
	if err := json.Unmarshal([]byte(value), &node); err != nil {
		return fmt.Errorf("failed to unmarshal node: %v", err)
	}

	node.Status = status
	return SaveNode(node)
}

func ListNodes() []models.Node {
	if own_redis.RedisClient == nil {
		fmt.Printf("❌ RedisClient is not initialized\n")
		return nil
	}

	pattern := "nodes:*"
	keys, err := own_redis.RedisClient.Keys(own_redis.Ctx, pattern).Result()
	if err != nil {
		fmt.Printf("❌ Failed to list nodes: %v\n", err)
		return nil
	}

	var nodes []models.Node
	for _, key := range keys {
		value, err := own_redis.RedisClient.Get(own_redis.Ctx, key).Result()
		if err != nil {
			fmt.Printf("❌ Failed to get node: %v\n", err)
			continue
		}

		var node models.Node
		if err := json.Unmarshal([]byte(value), &node); err != nil {
			fmt.Printf("❌ Failed to unmarshal node: %v\n", err)
			continue
		}

		nodes = append(nodes, node)
	}

	return nodes
}

func PublishEvent(eventType, podName string) {
	channel := "pods:events"
	message := fmt.Sprintf("%s:%s", eventType, podName)
	err := own_redis.RedisClient.Publish(own_redis.Ctx, channel, message).Err()
	if err != nil {
		fmt.Printf("❌ Failed to publish event: %v\n", err)
	}
}

func WatchPods() {
	sub := own_redis.RedisClient.Subscribe(own_redis.Ctx, "pods:events")
	defer sub.Close()

	for msg := range sub.Channel() {
		fmt.Printf("🔄 Event received: %s\n", msg.Payload)
	}
}

func generateNodePort() int {
	// NodePort range is typically 30000-32767
	min := 30000
	max := 32767

	// Get existing services to check used ports
	services := ListServices("")
	usedPorts := make(map[int]bool)

	for _, svc := range services {
		for _, port := range svc.Spec.Ports {
			if port.NodePort != 0 {
				usedPorts[port.NodePort] = true
			}
		}
	}

	// Find first available port
	for port := min; port <= max; port++ {
		if !usedPorts[port] {
			return port
		}
	}

	return min // Fallback to minimum port
}

func ListServices(namespace string) []models.Service {
	if namespace == "" {
		namespace = "default"
	}

	pattern := fmt.Sprintf("services:%s:*", namespace)
	keys, err := own_redis.RedisClient.Keys(own_redis.Ctx, pattern).Result()
	if err != nil {
		fmt.Printf("❌ Failed to list services: %v\n", err)
		return nil
	}

	var services []models.Service
	for _, key := range keys {
		value, err := own_redis.RedisClient.Get(own_redis.Ctx, key).Result()
		if err != nil {
			fmt.Printf("❌ Error getting service for key '%s': %v\n", key, err)
			continue
		}

		var service models.Service
		if err := json.Unmarshal([]byte(value), &service); err != nil {
			fmt.Printf("❌ Error unmarshaling service for key '%s': %v\n", key, err)
			continue
		}
		services = append(services, service)
	}

	fmt.Printf("✅ Found %d services in namespace '%s'\n", len(services), namespace)
	return services
}

func findServicesForPod(pod *models.Pod) []models.Service {
	if pod.Metadata.Labels == nil {
		return nil
	}

	services := ListServices(pod.Metadata.Namespace)
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
