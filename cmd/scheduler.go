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
				continue
			}

			// 1. Node seç
			selected := nodes[rand.Intn(len(nodes))]
			pod.Spec.NodeName = selected.Name
			pod.Status.HostIP = selected.IP

			// 2. Container çalıştır
			container := pod.Spec.Containers[0]
			containerName := fmt.Sprintf("%s-%s", pod.Metadata.Name, container.Name)

			args := []string{"run", "--rm", "-d", "--name", containerName, container.Image}
			if len(container.Cmd) > 0 {
				args = append(args, container.Cmd...)
			}

			err := exec.Command("docker", args...).Run()

			if err != nil {
				fmt.Printf("❌ Failed to start container for pod '%s': %v\n", pod.Metadata.Name, err)
				pod.Status.Phase = "Failed"
			} else {
				pod.Status.Phase = "Running"
				fmt.Printf("✅ Scheduled and started pod '%s' on node '%s'\n", pod.Metadata.Name, selected.Name)
			}

			store.SavePod(pod)
		}
	},
}

func init() {
	rootCmd.AddCommand(schedulerCmd)
}
