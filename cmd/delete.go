package cmd

import (
	"fmt"

	"github.com/selimhanmrl/Own-Kubernetes/store"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a pod by its name",
	Args:  cobra.ExactArgs(1), // Ensure exactly one argument is provided
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if store.DeletePodByName(name) {
			fmt.Printf("✅ Pod with name '%s' deleted successfully.\n", name)
		} else {
			fmt.Printf("❌ Pod with name '%s' not found.\n", name)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
