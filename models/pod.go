// models/pod.go
package models

type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	UID       string `json:"uid"`
}

type Container struct {
	Name  string   `json:"name"`
	Image string   `json:"image"`
	Cmd   []string `json:"cmd"`
}

type PodSpec struct {
	Containers []Container `json:"containers"`
	NodeName   string      `json:"nodeName,omitempty"` // empty until scheduled
}

type PodStatus struct {
	Phase     string `json:"phase"` // Pending, Running, Failed
	HostIP    string `json:"hostIP"`
	PodIP     string `json:"podIP"`
	StartTime string `json:"startTime"`
}

type Pod struct {
	Metadata Metadata  `json:"metadata"`
	Spec     PodSpec   `json:"spec"`
	Status   PodStatus `json:"status"`
}
