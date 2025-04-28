package models

type Service struct {
    Name      string            `json:"name" yaml:"name"`
    Namespace string            `json:"namespace" yaml:"namespace"`
    Selector  map[string]string `json:"selector" yaml:"selector"` // Match pods by labels
    Type      string            `json:"type" yaml:"type"`         // ClusterIP or NodePort
    Ports     []ServicePort     `json:"ports" yaml:"ports"`       // List of service ports
}

type ServicePort struct {
    Port       int `json:"port" yaml:"port"`             // Service port
    TargetPort int `json:"targetPort" yaml:"targetPort"` // Pod port
    NodePort   int `json:"nodePort" yaml:"nodePort"`     // Node port (for NodePort services)
}