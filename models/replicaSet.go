package models

type ReplicaSet struct {

    Metadata   ReplicaSetMetadata `yaml:"metadata"`
    Spec       ReplicaSetSpec     `yaml:"spec"`
    Status     ReplicaSetStatus   `yaml:"status"`
}

type ReplicaSetMetadata struct {
    Name        string            `yaml:"name"`
    Namespace   string            `yaml:"namespace"`
    Labels      map[string]string `yaml:"labels,omitempty"`
    Annotations map[string]string `yaml:"annotations,omitempty"`
    UID         string            `yaml:"uid,omitempty"`
}

type ReplicaSetSpec struct {
    Replicas int          `yaml:"replicas"`
    Selector LabelSelector `yaml:"selector"`
    Template PodTemplate  `yaml:"template"`
}

type LabelSelector struct {
    MatchLabels map[string]string `yaml:"matchLabels,omitempty"`
}

type ReplicaSetStatus struct {
    Replicas           int   `yaml:"replicas"`
    ReadyReplicas      int   `yaml:"readyReplicas,omitempty"`
    AvailableReplicas  int   `yaml:"availableReplicas,omitempty"`
    ObservedGeneration int64 `yaml:"observedGeneration,omitempty"`
}

// PodTemplate represents the template for creating new pods
type PodTemplate struct {
    Metadata PodTemplateMetadata `yaml:"metadata"`
    Spec     PodSpec            `yaml:"spec"`
}

// PodTemplateMetadata contains metadata for pod template
type PodTemplateMetadata struct {
    Labels      map[string]string `yaml:"labels,omitempty"`
    Annotations map[string]string `yaml:"annotations,omitempty"`
}