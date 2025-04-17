package cmd

import (
	"fmt"

	"github.com/selimhanmrl/Own-Kubernetes/store"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [pod-name]",
	Short: "Delete a pod by its name",
	Args:  cobra.ExactArgs(1), // Ensure exactly one argument is provided
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		if namespace == "" {
			namespace = "default" // Default to 'default' namespace
		}

		if store.DeletePodByName(name, namespace) {
			fmt.Printf("✅ Pod with name '%s' deleted successfully from namespace '%s'.\n", name, namespace)
		} else {
			fmt.Printf("❌ Pod with name '%s' not found in namespace '%s'.\n", name, namespace)
		}
	},
}

func init() {
	// Add namespace flag to the delete command
	deleteCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of the pod")
	rootCmd.AddCommand(deleteCmd)
}
