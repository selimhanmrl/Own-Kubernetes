package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/selimhanmrl/Own-Kubernetes/models"
)

var (
	mu        sync.Mutex
	storeFile = "pods.json"
	nodeStore = []models.Node{
		{Name: "node1", IP: "192.168.1.10"},
	}

	RedisClient *redis.Client
	ctx         = context.Background()
)

func SavePod(pod models.Pod) {
	key := fmt.Sprintf("pods:%s", pod.Metadata.UID)
	value, _ := json.Marshal(pod)

	err := RedisClient.Set(ctx, key, value, 0).Err()
	if err != nil {
		fmt.Printf("❌ Failed to save pod '%s': %v\n", pod.Metadata.Name, err)
		return
	}
	fmt.Printf("✅ Pod '%s' saved to Redis.\n", pod.Metadata.Name)
	// Publish an event after saving the pod
	PublishEvent("create", pod.Metadata.Name)
}

func GetPod(uid string) (models.Pod, bool) {
	key := fmt.Sprintf("pods:%s", uid)
	value, err := RedisClient.Get(ctx, key).Result()
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
	err := RedisClient.Del(ctx, key).Err()
	if err != nil {
		fmt.Printf("❌ Failed to delete pod '%s': %v\n", name, err)
		return false
	}

	fmt.Printf("✅ Pod '%s' deleted successfully.\n", name)
	return true
}

func ListPods() []models.Pod {
	keys, err := RedisClient.Keys(ctx, "pods:*").Result()
	if err != nil {
		fmt.Printf("❌ Failed to list pods: %v\n", err)
		return nil
	}

	var pods []models.Pod
	for _, key := range keys {
		value, _ := RedisClient.Get(ctx, key).Result()
		var pod models.Pod
		json.Unmarshal([]byte(value), &pod)
		pods = append(pods, pod)
	}
	return pods
}

// DeletePodByName deletes a pod by its name and stops the corresponding Docker container if it exists.
func DeletePod(uid string) bool {
	key := fmt.Sprintf("pods:%s", uid)
	err := RedisClient.Del(ctx, key).Err()
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
	err := RedisClient.Publish(ctx, channel, message).Err()
	if err != nil {
		fmt.Printf("❌ Failed to publish event: %v\n", err)
	}
}

func WatchPods() {
	sub := RedisClient.Subscribe(ctx, "pods:events")
	defer sub.Close()

	for msg := range sub.Channel() {
		fmt.Printf("🔄 Event received: %s\n", msg.Payload)
	}
}

// ---------------------------
// Internal JSON I/O Helpers
// ---------------------------

func loadAll() map[string]models.Pod {
	data, err := os.ReadFile(storeFile)
	if err != nil || len(data) == 0 { // Handle missing or empty file
		return make(map[string]models.Pod) // Initialize an empty map
	}

	var pods map[string]models.Pod
	err = json.Unmarshal(data, &pods)
	if err != nil { // Handle invalid JSON
		return make(map[string]models.Pod) // Initialize an empty map
	}
	return pods
}

func saveAll(pods map[string]models.Pod) {
	data, _ := json.MarshalIndent(pods, "", "  ")
	_ = os.WriteFile(storeFile, data, 0644)
}

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // No password by default
		DB:       0,                // Default DB
	})

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}
	log.Println("✅ Connected to Redis")
}
