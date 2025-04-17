package cmd

import (
    "fmt"
    "os/exec"

    "github.com/selimhanmrl/Own-Kubernetes/models"
    "github.com/selimhanmrl/Own-Kubernetes/store"
    "github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
    Use:   "logs [pod-name]",
    Short: "Fetch logs for a specific pod",
    Args:  cobra.ExactArgs(1), // Ensure exactly one argument is provided
    Run: func(cmd *cobra.Command, args []string) {
        podName := args[0]

        if namespace == "" {
            namespace = "default" // Default to 'default' namespace
        }

        // Find the pod by name in the specified namespace
        pods := store.ListPods(namespace)
        var podFound bool
        var pod models.Pod
        for _, p := range pods {
            if p.Metadata.Name == podName {
                pod = p
                podFound = true
                break
            }
        }

        if !podFound {
            fmt.Printf("❌ Pod with name '%s' not found in namespace '%s'.\n", podName, namespace)
            return
        }

        // Generate the container name
        containerName := fmt.Sprintf("%s-%s-%s", pod.Metadata.Name, pod.Spec.Containers[0].Name, pod.Metadata.UID[:8])

        // Fetch logs using `docker logs`
        out, err := exec.Command("docker", "logs", containerName).CombinedOutput()
        if err != nil {
            fmt.Printf("❌ Failed to fetch logs for pod '%s': %v\n", podName, err)
            return
        }

        fmt.Printf("📄 Logs for pod '%s' in namespace '%s':\n", podName, namespace)
        fmt.Println(string(out))
    },
}

func init() {
    // Add namespace flag to the logs command
    logsCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of the pod")
    rootCmd.AddCommand(logsCmd)
}