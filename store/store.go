package store

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/selimhanmrl/Own-Kubernetes/models"
)

var (
	mu        sync.Mutex
	storeFile = "pods.json"
	nodeStore = []models.Node{
		{Name: "node1", IP: "192.168.1.10"},
		{Name: "node2", IP: "192.168.1.11"},
	}
)

func SavePod(pod models.Pod) {
	mu.Lock()
	defer mu.Unlock()

	pods := loadAll()
	pods[pod.Metadata.UID] = pod
	saveAll(pods)
}

func GetPod(uid string) (models.Pod, bool) {
	pods := loadAll()
	pod, found := pods[uid]
	return pod, found
}

func ListPods() []models.Pod {
	pods := loadAll()
	var list []models.Pod
	for _, pod := range pods {
		list = append(list, pod)
	}
	return list
}
func DeletePodByName(name string) bool {
	mu.Lock()
	defer mu.Unlock()

	pods := loadAll()

	// Find the pod by name
	var uid string
	var pod models.Pod
	found := false
	for id, p := range pods {
		if p.Metadata.Name == name {
			uid = id
			pod = p
			found = true
			break
		}
	}

	if !found {
		return false // Pod not found
	}

	// Check pod status before stopping the Docker container
	if pod.Status.Phase != "Pending" && pod.Status.Phase != "Failed" {
		// Stop the Docker container
		containerName := fmt.Sprintf("%s-%s", pod.Metadata.Name, pod.Spec.Containers[0].Name)
		err := exec.Command("docker", "stop", containerName).Run()
		if err != nil {
			fmt.Printf("❌ Failed to stop container '%s': %v\n", containerName, err)
		}
	} else {
		fmt.Printf("⚠️ Pod '%s' is in '%s' state. Skipping Docker stop.\n", pod.Metadata.Name, pod.Status.Phase)
	}

	// Remove the pod from the map
	delete(pods, uid)
	saveAll(pods) // Save the updated map to the file
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
