package cmd

import (
	"fmt"

	"github.com/selimhanmrl/Own-Kubernetes/client"
	"github.com/spf13/cobra"
)

func getClient() *client.Client {
	return client.NewClient(client.ClientConfig{
		Host: apiHost,
		Port: apiPort,
	})
}

var deleteCmd = &cobra.Command{
	Use:   "delete [resource] [name]",
	Short: "Delete a resource",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		resourceType := args[0]
		name := args[1]

		client := getClient()

		switch resourceType {
		case "pod":
			if err := client.DeletePod("default", name); err != nil {
				fmt.Printf("❌ Failed to delete pod: %v\n", err)
				return
			}
			fmt.Printf("✅ Pod '%s' deleted successfully\n", name)
		default:
			fmt.Printf("❌ Unknown resource type: %s\n", resourceType)
		}
	},
}

func init() {
	// Add api-host and api-port flags
	deleteCmd.Flags().StringVar(&apiHost, "api-host", "localhost", "API server hostname")
	deleteCmd.Flags().StringVar(&apiPort, "api-port", "8080", "API server port")
	deleteCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of the pod")
	rootCmd.AddCommand(deleteCmd)
}
