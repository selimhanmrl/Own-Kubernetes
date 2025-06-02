package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/selimhanmrl/Own-Kubernetes/client"
	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/spf13/cobra"
)

var (
	nodeIP     string
	nodeLabels string
)

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Node management commands",
	Long:  `Create, list, and manage nodes in the cluster`,
}

var registerNodeCmd = &cobra.Command{
	Use:   "register [node-name]",
	Short: "Register a new node",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nodeName := args[0]

		// Create client
		c := client.NewClient(client.ClientConfig{
			Host: apiHost,
			Port: apiPort,
		})

		// Create node object
		node := models.Node{
			Name: nodeName,
			IP:   nodeIP,
			Status: models.NodeStatus{
				Phase:         "Ready",
				LastHeartbeat: time.Now(),
				Conditions: []models.NodeCondition{
					{
						Type:           "Ready",
						Status:         "True",
						LastUpdateTime: time.Now(),
					},
				},
			},
		}

		// Register node
		if err := c.RegisterNode(node); err != nil {
			fmt.Printf("❌ Failed to register node: %v\n", err)
			return
		}

		fmt.Printf("✅ Node '%s' registered successfully\n", nodeName)
	},
}

var getNodesCmd = &cobra.Command{
	Use:   "get nodes",
	Short: "List all nodes",
	Run: func(cmd *cobra.Command, args []string) {
		c := client.NewClient(client.ClientConfig{
			Host: apiHost,
			Port: apiPort,
		})

		nodes, err := c.ListNodes()
		if err != nil {
			fmt.Printf("Failed to list nodes: %v\n", err)
			return
		}

		if len(nodes) == 0 {
			fmt.Println("No nodes found.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tIP\tPODS\tAGE")

		for _, node := range nodes {
			age := "unknown"
			if !node.Status.LastHeartbeat.IsZero() {
				age = time.Since(node.Status.LastHeartbeat).Round(time.Second).String()
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
				node.Name,
				node.Status.Phase,
				node.IP,
				len(node.Pods),
				age,
			)
		}

		w.Flush()
	},
}

var createNodeCmd = &cobra.Command{
	Use:   "create [node-name]",
	Short: "Register and start a new node server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nodeName := args[0]

		// Create client
		c := client.NewClient(client.ClientConfig{
			Host: apiHost,
			Port: apiPort,
		})

		// Create and register node
		node := models.Node{
			Name: nodeName,
			IP:   nodeIP,
			Status: models.NodeStatus{
				Phase:         "Ready",
				LastHeartbeat: time.Now(),
				Conditions: []models.NodeCondition{
					{
						Type:           "Ready",
						Status:         "True",
						LastUpdateTime: time.Now(),
					},
				},
			},
		}

		// Parse and add labels if provided
		if nodeLabels != "" {
			labels := make(map[string]string)
			for _, label := range strings.Split(nodeLabels, ",") {
				parts := strings.Split(label, "=")
				if len(parts) == 2 {
					labels[parts[0]] = parts[1]
				}
			}
			node.Labels = labels
		}

		// Register node with API server
		if err := c.RegisterNode(node); err != nil {
			fmt.Printf("❌ Failed to register node: %v\n", err)
			return
		}

		fmt.Printf("✅ Node '%s' registered successfully\n", nodeName)
		fmt.Printf("📝 To start the node server, run this command on the target machine:\n")
		fmt.Printf("    go run . node-server %s --api-host %s --api-port %s --node-ip %s\n",
			nodeName, apiHost, apiPort, nodeIP)
	},
}

func init() {
	// Remove Docker-specific flags
	createNodeCmd.Flags().StringVar(&nodeIP, "ip", "", "IP address of the node")
	createNodeCmd.Flags().StringVar(&nodeLabels, "labels", "", "Labels for the node (comma-separated key=value pairs)")
	createNodeCmd.MarkFlagRequired("ip")

	nodeCmd.AddCommand(createNodeCmd)
	nodeCmd.AddCommand(getNodesCmd)
	rootCmd.AddCommand(nodeCmd)
}
