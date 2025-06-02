package main

import (
	"fmt"
	"log"
	"os"

	"github.com/selimhanmrl/Own-Kubernetes/agent"
	"github.com/selimhanmrl/Own-Kubernetes/client"
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

	if len(args) > 0 && args[0] == "proxy" {
		fmt.Println("🚀 Starting kube-proxy...")
		proxyServer := server.NewProxyServer()

		// Get services and pods
		client := client.NewClient(client.ClientConfig{
			Host: "localhost",
			Port: "8080",
		})

		services, err := client.ListServices("")
		if err != nil {
			fmt.Printf("❌ Failed to list services: %v\n", err)
			os.Exit(1)
		}

		// Register NodePort services
		for _, svc := range services {
			if svc.Spec.Type == "NodePort" {
				pods, err := client.ListPods("")
				if err != nil {
					fmt.Printf("❌ Failed to list pods: %v\n", err)
					continue
				}

				fmt.Printf("📦 Registering service %s with NodePort\n", svc.Metadata.Name)
				proxyServer.RegisterService(&svc, pods)
			}
		}

		// Start the proxy server
		if err := proxyServer.Start(); err != nil {
			fmt.Printf("❌ Failed to start proxy: %v\n", err)
			os.Exit(1)
		}

		// Keep the proxy running
		select {}
	}

	// If no mode specified, assume it's a CLI command
	if err := cmd.Execute(); err != nil {
		log.Fatalf("❌ Error executing CLI command: %v", err)
	}
}
