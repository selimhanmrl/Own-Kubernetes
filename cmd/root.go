package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mykube",
	Short: "MyKube is a tiny container orchestration CLI",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(getCmd)
}
