package main

import (
	"log"
	"os"

	"github.com/selimhanmrl/Own-Kubernetes/agent"
	"github.com/selimhanmrl/Own-Kubernetes/cmd"
	own_redis "github.com/selimhanmrl/Own-Kubernetes/redis"
	"github.com/selimhanmrl/Own-Kubernetes/server"
)

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "-mode" {
		mode := "server"
		if len(args) > 1 {
			mode = args[1]
		}

		switch mode {
		case "server":
			// Initialize Redis
			own_redis.InitRedis()

			// Create and start API server
			apiServer := server.NewAPIServer()
			apiServer.Start()

		case "node":
			// Start as a node
			nodeName := os.Getenv("NODE_NAME")
			nodeIP := os.Getenv("NODE_IP")
			apiHost := os.Getenv("API_HOST")
			apiPort := os.Getenv("API_PORT")

			if nodeName == "" || nodeIP == "" {
				log.Fatal("❌ NODE_NAME and NODE_IP environment variables are required")
			}

			if apiHost == "" {
				apiHost = "localhost"
			}
			if apiPort == "" {
				apiPort = "8080"
			}

			nodeAgent := agent.NewNodeAgent(nodeName, nodeIP, apiHost, apiPort)
			if err := nodeAgent.Start(); err != nil {
				log.Fatalf("❌ Failed to start node agent: %v", err)
			}

			// Keep the node running
			select {}

		case "cli":
			// Remove the -mode cli arguments before passing to cobra
			os.Args = append(os.Args[:1], os.Args[3:]...)
			if err := cmd.Execute(); err != nil {
				log.Fatalf("❌ Error executing CLI command: %v", err)
			}

		default:
			log.Fatalf("❌ Invalid mode: %s. Must be 'server', 'node', or 'cli'", mode)
		}
		return
	}

	// If no mode specified, assume it's a CLI command
	if err := cmd.Execute(); err != nil {
		log.Fatalf("❌ Error executing CLI command: %v", err)
	}
}
