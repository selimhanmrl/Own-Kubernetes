package models

import "time"

type NodeStatus struct {
	Conditions    []NodeCondition `json:"conditions"`
	Capacity      ResourceList    `json:"capacity"`
	Allocatable   ResourceList    `json:"allocatable"`
	Phase         string          `json:"phase"` // Ready, NotReady
	LastHeartbeat time.Time       `json:"lastHeartbeat"`
}

type NodeCondition struct {
	Type           string    `json:"type"`   // Ready, DiskPressure, MemoryPressure, NetworkUnavailable
	Status         string    `json:"status"` // True, False, Unknown
	LastUpdateTime time.Time `json:"lastUpdateTime"`
}

type ResourceList map[string]string

type Node struct {
	Name   string            `json:"name"`
	IP     string            `json:"ip"`
	Labels map[string]string `json:"labels,omitempty"`
	Status NodeStatus        `json:"status"`
	Pods   []string          `json:"pods"` // List of pod UIDs running on this node
}

type NodeResources struct {
    CPU    string `json:"cpu"`
    Memory string `json:"memory"`
    Pods   string `json:"pods"`
}

type NodeSpec struct {
    IP        string            `json:"ip"`
    Resources NodeResources     `json:"resources"`
    Labels    map[string]string `json:"labels,omitempty"`
}
