// store/store.go
package store

import (
    "sync"
    "github.com/selimhanmrl/Own-Kubernetes/models"
)

var (
    podStore = make(map[string]models.Pod)
    mu       sync.Mutex
)

func SavePod(pod models.Pod) {
    mu.Lock()
    defer mu.Unlock()
    podStore[pod.Metadata.UID] = pod
}

func GetPod(uid string) (models.Pod, bool) {
    mu.Lock()
    defer mu.Unlock()
    pod, found := podStore[uid]
    return pod, found
}

func ListPods() []models.Pod {
    mu.Lock()
    defer mu.Unlock()
    pods := []models.Pod{}
    for _, pod := range podStore {
        pods = append(pods, pod)
    }
    return pods
}
