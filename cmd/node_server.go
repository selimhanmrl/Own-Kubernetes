package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/selimhanmrl/Own-Kubernetes/server"
)

var (
    nodePort string
)

var nodeServerCmd = &cobra.Command{
    Use:   "node-server [node-name]",
    Short: "Start a node server",
    Long:  `Start a node server that manages containers on this machine`,
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        nodeName := args[0]
        
        // Create node server with all parameters
        nodeServer := server.NewNodeServer(
            nodeName,
            nodePort,
            nodeIP,
            apiHost,
            apiPort,
        )
        
        fmt.Printf("Starting node server %s on %s:%s\n", nodeName, nodeIP, nodePort)
        return nodeServer.Start()
    },
}

func init() {
    nodeServerCmd.Flags().StringVar(&nodePort, "port", "8081", "Port for the node server")
    nodeServerCmd.Flags().StringVar(&nodeIP, "node-ip", "", "IP address of this node")
    nodeServerCmd.MarkFlagRequired("node-ip")
}