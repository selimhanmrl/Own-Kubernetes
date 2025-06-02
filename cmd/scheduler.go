package cmd

import (
	"fmt"
	"time"

	"github.com/selimhanmrl/Own-Kubernetes/client"
	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/spf13/cobra"
)

var schedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Run the scheduler to assign pods to nodes",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🎯 Starting scheduler...")

		c := client.NewClient(client.ClientConfig{
			Host: apiHost,
			Port: apiPort,
		})

		// Wait for API server to be ready
		fmt.Println("⌛ Waiting for API server...")
		for {
			_, err := c.ListNodes()
			if err == nil {
				break
			}
			fmt.Printf("🔄 Retrying connection to API server...\n")
			time.Sleep(2 * time.Second)
		}

		fmt.Println("✅ Connected to API server")

		for {
			pods, err := c.ListPods("")
			if err != nil {
				fmt.Printf("❌ Failed to list pods: %v\n", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// Schedule pending pods
			for _, pod := range pods {
				if pod.Status.Phase != "Pending" || pod.Spec.NodeName != "" {
					continue
				}

				if err := assignNodeToPod(&pod, c); err != nil {
					fmt.Printf("❌ Failed to assign node to pod '%s': %v\n",
						pod.Metadata.Name, err)
					continue
				}

				fmt.Printf("✅ Successfully assigned pod '%s' to node '%s'\n",
					pod.Metadata.Name, pod.Spec.NodeName)
			}

			time.Sleep(5 * time.Second)
		}
	},
}

var lastNodeIndex = 0

func assignNodeToPod(pod *models.Pod, c *client.Client) error {
	// Get nodes from API server instead of store
	nodes, err := c.ListNodes()
	if err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}

	if len(nodes) == 0 {
		return fmt.Errorf("no nodes available")
	}

	// Filter ready nodes
	readyNodes := []models.Node{}
	for _, node := range nodes {
		if node.Status.Phase == "Ready" {
			readyNodes = append(readyNodes, node)
		}
	}

	if len(readyNodes) == 0 {
		return fmt.Errorf("no ready nodes available")
	}

	// Round-robin selection
	selectedNode := readyNodes[lastNodeIndex%len(readyNodes)]
	lastNodeIndex++

	fmt.Printf("🔄 Round-robin selected node '%s' (index: %d)\n",
		selectedNode.Name, lastNodeIndex-1)

	// Assign node to pod
	pod.Spec.NodeName = selectedNode.Name
	pod.Status.HostIP = selectedNode.IP

	// Update pod through API
	return c.UpdatePod(*pod)
}

func init() {
	// Add configuration flags
	schedulerCmd.Flags().StringVar(&apiHost, "api-host", "localhost", "API server hostname")
	schedulerCmd.Flags().StringVar(&apiPort, "api-port", "8080", "API server port")
	schedulerCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to filter services and pods")
	rootCmd.AddCommand(schedulerCmd)
}
