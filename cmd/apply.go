package cmd

import (
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/google/uuid"
    "github.com/selimhanmrl/Own-Kubernetes/models"
    "github.com/selimhanmrl/Own-Kubernetes/store"
    "github.com/spf13/cobra"
    "gopkg.in/yaml.v3"
)

var file string

var applyCmd = &cobra.Command{
    Use:   "apply",
    Short: "Apply a YAML resource definition",
    Run: func(cmd *cobra.Command, args []string) {
        data, err := os.ReadFile(file)
        if err != nil {
            fmt.Println("❌ Failed to read file:", err)
            return
        }

        // First, unmarshal to get the resource kind
        var resource struct {
            Kind string `yaml:"kind"`
        }
        if err := yaml.Unmarshal(data, &resource); err != nil {
            fmt.Println("❌ Failed to parse YAML:", err)
            return
        }

        switch resource.Kind {
        case "Pod":
            applyPod(data)
        case "ReplicaSet":
            applyReplicaSet(data)
        default:
            fmt.Printf("❌ Unsupported resource kind: %s\n", resource.Kind)
        }
    },
}

func applyPod(data []byte) {
    var pod models.Pod
    if err := yaml.Unmarshal(data, &pod); err != nil {
        fmt.Println("❌ Failed to parse Pod YAML:", err)
        return
    }

    // Set namespace if not provided in the YAML
    if pod.Metadata.Namespace == "" {
        if namespace == "" {
            namespace = "default"
        }
        pod.Metadata.Namespace = namespace
    }

    originalName := pod.Metadata.Name
    pod.Metadata.UID = uuid.NewString()
    pod.Status.Phase = "Pending"
    pod.Status.StartTime = time.Now().Format(time.RFC3339)
    pod.Metadata.Name = fmt.Sprintf("%s-%s-%s", originalName, pod.Metadata.UID[:4], pod.Metadata.UID[4:8])
    
    if err := store.SavePod(pod); err != nil {
        fmt.Printf("❌ Failed to save pod: %v\n", err)
        return
    }

    out, _ := json.MarshalIndent(pod, "", "  ")
    fmt.Println("✅ Pod created:")
    fmt.Println(string(out))
}

func applyReplicaSet(data []byte) {
    var rs models.ReplicaSet
    if err := yaml.Unmarshal(data, &rs); err != nil {
        fmt.Println("❌ Failed to parse ReplicaSet YAML:", err)
        return
    }

    // Set namespace if not provided
    if rs.Metadata.Namespace == "" {
        if namespace == "" {
            namespace = "default"
        }
        rs.Metadata.Namespace = namespace
    }

    fmt.Printf("📦 Creating ReplicaSet '%s' with %d replicas...\n", rs.Metadata.Name, rs.Spec.Replicas)

    // Create pods for each replica
    for i := 0; i < rs.Spec.Replicas; i++ {
        pod := models.Pod{
            Metadata: models.Metadata{
                Name:      fmt.Sprintf("%s-%d", rs.Metadata.Name, i+1),
                Namespace: rs.Metadata.Namespace,
                Labels:    rs.Spec.Template.Metadata.Labels,
                UID:      uuid.NewString(),
            },
            Spec:   rs.Spec.Template.Spec,
            Status: models.PodStatus{
                Phase:     "Pending",
                StartTime: time.Now().Format(time.RFC3339),
            },
        }

        if err := store.SavePod(pod); err != nil {
            fmt.Printf("❌ Failed to create replica %d: %v\n", i+1, err)
            continue
        }

        fmt.Printf("✅ Created replica %d/%d: %s\n", i+1, rs.Spec.Replicas, pod.Metadata.Name)
    }

    // Save the ReplicaSet itself
    rs.Metadata.UID = uuid.NewString()
    if err := store.SaveReplicaSet(rs); err != nil {
        fmt.Printf("❌ Failed to save ReplicaSet: %v\n", err)
        return
    }

    fmt.Printf("✅ ReplicaSet '%s' created successfully\n", rs.Metadata.Name)
}

func init() {
    applyCmd.Flags().StringVarP(&file, "file", "f", "", "YAML file to apply")
    applyCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to apply the resource to")
    applyCmd.MarkFlagRequired("file")
    rootCmd.AddCommand(applyCmd)
}