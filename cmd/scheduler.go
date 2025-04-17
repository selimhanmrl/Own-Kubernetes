package cmd

import (
	"fmt"
	"os/exec"

	"github.com/selimhanmrl/Own-Kubernetes/store"
	"github.com/spf13/cobra"
)

var schedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Run the scheduler to assign pods to nodes and run containers",
	Run: func(cmd *cobra.Command, args []string) {
		if namespace == "" {
			namespace = "default" // Default to 'default' namespace
		}

		// List pods in the specified namespace
		pods := store.ListPods(namespace)
		if len(pods) == 0 {
			fmt.Printf("No pods found in namespace '%s'.\n", namespace)
			return
		}

		for _, pod := range pods {
			if pod.Status.Phase != "Pending" {
				fmt.Printf("⚠️ Pod '%s' is in '%s' state. Skipping scheduling.\n", pod.Metadata.Name, pod.Status.Phase)
				continue
			}

			// Select a node

			// Generate a unique container name
			containerName := fmt.Sprintf("%s-%s-%s", pod.Metadata.Name, pod.Spec.Containers[0].Name, pod.Metadata.UID[:8])

			args := []string{"run", "-d", "--name", containerName, pod.Spec.Containers[0].Image}
			if len(pod.Spec.Containers[0].Cmd) > 0 {
				args = append(args, pod.Spec.Containers[0].Cmd...)
			}

			// Start the Docker container and capture the container ID
			out, err := exec.Command("docker", args...).Output()
			if err != nil {
				fmt.Printf("❌ Failed to start container for pod '%s': %v\n", pod.Metadata.Name, err)
				pod.Status.Phase = "Failed"
			} else {
				pod.Status.Phase = "Running"
				pod.Status.ContainerID = string(out) // Store the container ID
				fmt.Printf("✅ Scheduled and started pod '%s' ", pod.Metadata.Name)
			}

			// Save the updated pod back to Redis
			store.SavePod(pod)
		}
	},
}

func init() {
	// Add namespace flag to the scheduler command
	schedulerCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace to filter pods")
	rootCmd.AddCommand(schedulerCmd)
}
