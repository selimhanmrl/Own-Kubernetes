package cmd

import (
	"fmt"
	"math/rand"
	"os/exec"

	"github.com/selimhanmrl/Own-Kubernetes/store"
	"github.com/spf13/cobra"
)

var schedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Run the scheduler to assign pods to nodes and run containers",
	Run: func(cmd *cobra.Command, args []string) {
		pods := store.ListPods()
		nodes := store.ListNodes()

		if len(nodes) == 0 {
			fmt.Println("❌ No nodes available")
			return
		}

		for _, pod := range pods {
			if pod.Status.Phase != "Pending" {
				fmt.Printf("⚠️ Pod '%s' is in '%s' state. Skipping scheduling.\n", pod.Metadata.Name, pod.Status.Phase)
				continue
			}
		
			// Select a node
			selected := nodes[rand.Intn(len(nodes))]
			pod.Spec.NodeName = selected.Name
			pod.Status.HostIP = selected.IP
		
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
				fmt.Printf("✅ Scheduled and started pod '%s' on node '%s'\n", pod.Metadata.Name, selected.Name)
			}
		
			store.SavePod(pod)
		}
	},
}

func init() {
	rootCmd.AddCommand(schedulerCmd)
}
