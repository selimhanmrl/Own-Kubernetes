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

func SavePod(pod models.Pod) {
	if own_redis.RedisClient == nil { // Use RedisClient from the redis package
		log.Fatalf("❌ RedisClient is not initialized")
	}

	key := fmt.Sprintf("pods:%s", pod.Metadata.UID)
	value, _ := json.Marshal(pod)

	err := own_redis.RedisClient.Set(own_redis.Ctx, key, value, 0).Err() // Use redis.Ctx
	if err != nil {
		fmt.Printf("❌ Failed to save pod '%s': %v\n", pod.Metadata.Name, err)
		return
	}
	fmt.Printf("✅ Pod '%s' saved to Redis.\n", pod.Metadata.Name)
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

func DeletePodByName(name string) bool {
	mu.Lock()
	defer mu.Unlock()

	// List all pods
	pods := ListPods()

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
		fmt.Printf("❌ Pod with name '%s' not found.\n", name)
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
	key := fmt.Sprintf("pods:%s", uid)
	err := own_redis.RedisClient.Del(own_redis.Ctx, key).Err() // Use redis.Ctx
	if err != nil {
		fmt.Printf("❌ Failed to delete pod '%s': %v\n", name, err)
		return false
	}

	fmt.Printf("✅ Pod '%s' deleted successfully.\n", name)
	return true
}

func ListPods() []models.Pod {
	keys, err := own_redis.RedisClient.Keys(own_redis.Ctx, "pods:*").Result()
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
func ListNodes() []models.Node {
	mu.Lock()
	defer mu.Unlock()
	nodes := []models.Node{}
	for _, node := range nodeStore {
		nodes = append(nodes, node)
	}
	return nodes
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
