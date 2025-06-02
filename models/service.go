package models

type Service struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   ServiceMetadata `yaml:"metadata"`
	Spec       ServiceSpec     `yaml:"spec"`
}

type ServiceMetadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

type ServiceSpec struct {
	Type     string            `yaml:"type"`
	Selector map[string]string `yaml:"selector"`
	Ports    []ServicePort     `yaml:"ports"`
}

type ServicePort struct {
	Port         int `yaml:"port"`
	TargetPort   int `yaml:"targetPort"`
	NodePort     int `yaml:"nodePort,omitempty"`
	assignedPort int // Internal field to track assigned port
}
