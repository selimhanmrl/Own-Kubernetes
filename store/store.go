package store

import (
	"encoding/json"
	"os"
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
