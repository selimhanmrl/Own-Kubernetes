package main

import (
	"log"
	"os"

	"github.com/selimhanmrl/Own-Kubernetes/agent"
)

func main() {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatal("NODE_NAME environment variable is required")
	}

	nodeIP := os.Getenv("NODE_IP")
	if nodeIP == "" {
		log.Fatal("NODE_IP environment variable is required")
	}

	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "localhost"
	}

	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	nodeAgent := agent.NewNodeAgent(nodeName, nodeIP, apiHost, apiPort)
	if err := nodeAgent.Start(); err != nil {
		log.Fatalf("Failed to start node agent: %v", err)
	}

	// Keep the agent running
	select {}
}
