package cmd

import (
	"fmt"
	"math"
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

		// Create client with Docker host connection
		c := client.NewClient(client.ClientConfig{
			Host: "localhost", // Connect to Docker host
			Port: "8080",      // API server exposed port
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

		// Get unscheduled pods
		pods, err := c.ListPods("")
		if err != nil {
			fmt.Printf("❌ Failed to list pods: %v\n", err)
			time.Sleep(5 * time.Second)
			return
		}

		// Get available nodes
		nodes, err := c.ListNodes()
		if err != nil {
			fmt.Printf("❌ Failed to list nodes: %v\n", err)
			time.Sleep(5 * time.Second)
			return
		}

		if len(nodes) == 0 {
			fmt.Println("⚠️ No nodes available for scheduling")
			time.Sleep(5 * time.Second)
			return
		}

		// Track node assignments for load balancing
		nodeAssignments := make(map[string]int)
		for _, node := range nodes {
			nodeAssignments[node.Name] = len(node.Pods)
		}

		// Schedule pending pods
		for _, pod := range pods {
			if pod.Status.Phase != "Pending" || pod.Spec.NodeName != "" {
				continue
			}

			// Find suitable node
			var selectedNode *models.Node
			minPods := math.MaxInt32

			for _, node := range nodes {
				if node.Status.Phase != "Ready" {
					continue
				}

				podsOnNode := nodeAssignments[node.Name]
				if podsOnNode < minPods {
					minPods = podsOnNode
					selectedNode = &node
				}
			}

			if selectedNode == nil {
				fmt.Printf("⚠️ No suitable node found for pod '%s'\n", pod.Metadata.Name)
				continue
			}

			fmt.Printf("📦 Scheduling pod '%s' to node '%s'...\n",
				pod.Metadata.Name, selectedNode.Name)

			// ONLY update pod assignment
			pod.Spec.NodeName = selectedNode.Name
			if err := c.UpdatePod(pod); err != nil {
				fmt.Printf("❌ Failed to update pod: %v\n", err)
				continue
			}

			nodeAssignments[selectedNode.Name]++
			fmt.Printf("✅ Successfully assigned pod '%s' to node '%s'\n",
				pod.Metadata.Name, selectedNode.Name)
		}

		time.Sleep(5 * time.Second) // Schedule check interval

	},
}

func init() {
	// Add configuration flags
	schedulerCmd.Flags().StringVar(&apiHost, "api-host", "localhost", "API server hostname")
	schedulerCmd.Flags().StringVar(&apiPort, "api-port", "8080", "API server port")
	schedulerCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to filter services and pods")
	rootCmd.AddCommand(schedulerCmd)
}
