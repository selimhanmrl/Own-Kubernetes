package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/selimhanmrl/Own-Kubernetes/store"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var serviceFile string

var applyServiceCmd = &cobra.Command{
	Use:   "apply-service",
	Short: "Apply a YAML service definition",
	Run: func(cmd *cobra.Command, args []string) {
		// Read the YAML file
		data, err := os.ReadFile(serviceFile)
		if err != nil {
			fmt.Println("❌ Failed to read file:", err)
			return
		}

		// Parse the YAML into the Service struct
		var service models.Service
		if err := yaml.Unmarshal(data, &service); err != nil {
			fmt.Println("❌ Failed to parse YAML:", err)
			return
		}

		// Set namespace if not provided in the YAML
		if service.Namespace == "" {
			if namespace == "" {
				namespace = "default" // Default to 'default' namespace
			}
			service.Namespace = namespace
		}

		// Validate and assign NodePorts
		usedPorts := getUsedNodePorts() // Get a list of already assigned NodePorts
		for i, port := range service.Ports {
			if service.Type == "NodePort" {
				if port.NodePort == 0 {
					// Assign a random NodePort if not specified
					service.Ports[i].NodePort = getRandomNodePort(usedPorts)
					usedPorts[service.Ports[i].NodePort] = true
					fmt.Printf("ℹ️ Assigned random NodePort %d for service '%s'.\n", service.Ports[i].NodePort, service.Name)
				} else {
					// Validate the specified NodePort
					if usedPorts[port.NodePort] {
						fmt.Printf("❌ NodePort %d is already in use. Cannot apply service '%s'.\n", port.NodePort, service.Name)
						return
					}
					usedPorts[port.NodePort] = true
				}
			}
		}

		// Debugging output
		fmt.Printf("Applying service '%s' in namespace '%s'...\n", service.Name, service.Namespace)
		fmt.Printf("Service Type: %s\n", service.Type)
		fmt.Printf("Service Ports: %+v\n", service.Ports)

		// Save the service to the store
		store.SaveService(service)
	},
}

func init() {
	applyServiceCmd.Flags().StringVarP(&serviceFile, "file", "f", "", "YAML file to apply")
	applyServiceCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to apply the service to")
	applyServiceCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(applyServiceCmd)
}

// getUsedNodePorts retrieves a map of all currently assigned NodePorts
func getUsedNodePorts() map[int]bool {
	usedPorts := make(map[int]bool)
	services := store.ListServices("") // List all services across all namespaces
	for _, service := range services {
		for _, port := range service.Ports {
			if service.Type == "NodePort" && port.NodePort > 0 {
				usedPorts[port.NodePort] = true
			}
		}
	}
	return usedPorts
}

// getRandomNodePort generates a random NodePort that is not already in use
func getRandomNodePort(usedPorts map[int]bool) int {
	rand.Seed(time.Now().UnixNano())
	for {
		port := rand.Intn(32767-30000+1) + 30000 // Generate a port in the range 30000-32767
		if !usedPorts[port] {
			return port
		}
	}
}
