package cmd

import (
    "github.com/spf13/cobra"
    "os"
)

var rootCmd = &cobra.Command{
    Use:   "mykube",
    Short: "MyKube is a CLI tool for managing Kubernetes resources",
    Long:  `MyKube is a command-line interface for applying, retrieving, and managing Kubernetes resources efficiently.`,
    Run: func(cmd *cobra.Command, args []string) {
        // Default action when no subcommands are provided
        cmd.Help()
    },
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func init() {
    // Define global flags and configurations here
}