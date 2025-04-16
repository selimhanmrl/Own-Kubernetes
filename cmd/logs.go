package cmd

import (
	"fmt"
	"os/exec"

	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/selimhanmrl/Own-Kubernetes/store"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Fetch logs for a specific pod",
	Args:  cobra.ExactArgs(1), // Ensure exactly one argument is provided
	Run: func(cmd *cobra.Command, args []string) {
		podName := args[0]

		// Find the pod by name
		pods := store.ListPods()
		var podFound bool
		var pod models.Pod // Use models.Pod instead of store.Pod
		for _, p := range pods {
			if p.Metadata.Name == podName {
				pod = p
				podFound = true
				break
			}
		}

		if !podFound {
			fmt.Printf("❌ Pod with name '%s' not found.\n", podName)
			return
		}

		// Generate the container name
		containerName := fmt.Sprintf("%s-%s", pod.Metadata.Name, pod.Spec.Containers[0].Name)

		// Fetch logs using `docker logs`
		out, err := exec.Command("docker", "logs", containerName).CombinedOutput()
		if err != nil {
			fmt.Printf("❌ Failed to fetch logs for pod '%s': %v\n", podName, err)
			return
		}

		fmt.Printf("📄 Logs for pod '%s':\n", podName)
		fmt.Println(string(out))
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
