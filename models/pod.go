// models/pod.go
package models

type Metadata struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	UID       string            `json:"uid"`
	Labels    map[string]string `json:"labels,omitempty"` // e.g., {"app": "nginx"}
}

type PodSpec struct {
	Containers []Container `json:"containers"`
	NodeName   string      `json:"nodeName,omitempty"` // empty until scheduled
	Replicas   int         `json:"replicas,omitempty"` // for deployment

}

type PodStatus struct {
	Phase       string `json:"phase"` // Pending, Running, Failed
	HostIP      string `json:"hostIP"`
	PodIP       string `json:"podIP"`
	StartTime   string `json:"startTime"`
	ContainerID string `json:"containerID"`
}

type Pod struct {
	Metadata Metadata  `json:"metadata"`
	Spec     PodSpec   `json:"spec"`
	Status   PodStatus `json:"status"`
}

type ResourceRequirements struct {
	Requests map[string]string `json:"requests"` // e.g., {"cpu": "250m", "memory": "64Mi"}
	Limits   map[string]string `json:"limits"`   // e.g., {"cpu": "500m", "memory": "128Mi"}
}

type Container struct {
	Name      string               `json:"name"`
	Image     string               `json:"image"`
	Cmd       []string             `json:"cmd"`
	Resources ResourceRequirements `json:"resources"` // Add resources field
}
