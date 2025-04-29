package store

import (
	"encoding/json"
	"fmt"
	"log"

	"os/exec"
	"sync"

	"github.com/go-redis/redis/v8"
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
	if own_redis.RedisClient == nil { // Use RedisClient from the redis package
		log.Fatalf("❌ RedisClient is not initialized")
	}

	if pod.Metadata.Namespace == "" {
		pod.Metadata.Namespace = "default" // Default to 'default' namespace
	}

	key := fmt.Sprintf("pods:%s:%s", pod.Metadata.Namespace, pod.Metadata.UID) // Include namespace in the key
	value, err := json.Marshal(pod)
	if err != nil {
		return fmt.Errorf("failed to marshal pod: %v", err)
	}

	err = own_redis.RedisClient.Set(own_redis.Ctx, key, value, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to save pod '%s': %v", pod.Metadata.Name, err)
	}

	fmt.Printf("✅ Pod '%s' saved to Redis in namespace '%s'.\n", pod.Metadata.Name, pod.Metadata.Namespace)
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

func ListServices(namespace string) []models.Service {
	if namespace == "" {
		namespace = "default" // Default to 'default' namespace
	}

	pattern := fmt.Sprintf("services:%s:*", namespace) // Match keys for the namespace
	keys, err := own_redis.RedisClient.Keys(own_redis.Ctx, pattern).Result()
	if err != nil {
		fmt.Printf("❌ Failed to list services: %v\n", err)
		return nil
	}

	var services []models.Service
	for _, key := range keys {
		value, _ := own_redis.RedisClient.Get(own_redis.Ctx, key).Result()
		var service models.Service
		json.Unmarshal([]byte(value), &service)
		services = append(services, service)
	}
	return services
}

func GetPod(uid string) (models.Pod, bool) {
	key := fmt.Sprintf("pods:%s", uid)
	value, err := own_redis.RedisClient.Get(own_redis.Ctx, key).Result() // Use redis.Ctx
	if err == redis.Nil {
		return models.Pod{}, false // Pod not found
	} else if err != nil {
		fmt.Printf("❌ Failed to get pod '%s': %v\n", uid, err)
		return models.Pod{}, false
	}

	var pod models.Pod
	json.Unmarshal([]byte(value), &pod)
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
		namespace = "default" // Default to 'default' namespace
	}

	pattern := fmt.Sprintf("pods:%s:*", namespace) // Match keys for the namespace
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

// DeletePodByName deletes a pod by its name and stops the corresponding Docker container if it exists.
func DeletePod(uid string) bool {
	key := fmt.Sprintf("pods:%s", uid)
	err := own_redis.RedisClient.Del(own_redis.Ctx, key).Err()
	if err != nil {
		fmt.Printf("❌ Failed to delete pod '%s': %v\n", uid, err)
		return false
	}
	fmt.Printf("✅ Pod '%s' deleted from Redis.\n", uid)
	// Stop the Docker container if it exists

	return true
}

func AddNode(node models.Node) {
	mu.Lock()
	defer mu.Unlock()
	nodeStore = append(nodeStore, node) // Append the new node to the slice
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
