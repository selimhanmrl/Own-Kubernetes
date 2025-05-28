package cmd

import (
	"fmt"
	"os"
	"os/exec"
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
	Short: "Create and start a new node container",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		nodeName := args[0]

		// Create a dedicated network if it doesn't exist
		exec.Command("docker", "network", "create", "own-k8s-net").Run()

		// Start the node container with isolated Docker daemon
		runCmd := exec.Command("docker", "run", "-d",
			"--name", nodeName,
			"--network", "own-k8s-net",
			"--hostname", nodeName, // Set hostname
			"--privileged",
			"--dns", "8.8.8.8", // Add Google DNS
			"--dns", "8.8.4.4", // Add backup DNS
			"-e", "DOCKER_TLS_CERTDIR=",
			"-e", fmt.Sprintf("NODE_NAME=%s", nodeName),
			"-e", fmt.Sprintf("NODE_IP=%s", nodeIP),
			"-e", fmt.Sprintf("API_HOST=api-server"),
			"-e", fmt.Sprintf("API_PORT=8080"), // Hardcode API port for now
			"--add-host", "api-server:172.19.0.2", // Add host entry
			"-v", "/var/run/docker.sock:/var/run/docker.sock", // Mount Docker socket
			"mykube-node") // Make sure this matches your image name

		output, err := runCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("❌ Failed to create node: %v\n%s\n", err, string(output))
			return
		}

		fmt.Printf("✅ Node '%s' created successfully\n", nodeName)
	},
}

func init() {
	registerNodeCmd.Flags().StringVar(&nodeIP, "ip", "", "IP address of the node")

	createNodeCmd.Flags().StringVar(&nodeIP, "ip", "", "IP address of the node")
	createNodeCmd.Flags().StringVar(&nodeLabels, "labels", "", "Labels for the node (comma-separated key=value pairs)")

	registerNodeCmd.MarkFlagRequired("ip")
	registerNodeCmd.Flags().StringVar(&nodeLabels, "labels", "", "Labels for the node (comma-separated key=value pairs)")

	nodeCmd.AddCommand(registerNodeCmd)
	nodeCmd.AddCommand(getNodesCmd)
	nodeCmd.AddCommand(createNodeCmd)
	rootCmd.AddCommand(nodeCmd)
}
