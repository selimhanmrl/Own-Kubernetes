package cmd

import (
	"github.com/spf13/cobra"
)

var (
	apiHost string
	apiPort string
)

var rootCmd = &cobra.Command{
	Use:   "mykube",
	Short: "MyKube is a tiny container orchestration CLI",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add global flags for API server configuration
	rootCmd.PersistentFlags().StringVar(&apiHost, "api-host", "localhost", "API server host")
	rootCmd.PersistentFlags().StringVar(&apiPort, "api-port", "8080", "API server port")

	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(nodeServerCmd) // Add this line
}
