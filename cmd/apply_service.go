package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var serviceFile string

var applyServiceCmd = &cobra.Command{
	Use:   "apply-service",
	Short: "Apply a YAML service definition",
	Run: func(cmd *cobra.Command, args []string) {
		// Get API client
		client := getClient()

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

		// Set namespace if not provided
		if service.Metadata.Namespace == "" {
			service.Metadata.Namespace = "default"
		}

		// Validate selector
		if len(service.Spec.Selector) == 0 {
			fmt.Println("❌ Service must have a selector")
			return
		}

		// Handle NodePort assignment
		if service.Spec.Type == "NodePort" {
			fmt.Println("📡 Processing NodePort service...")

			// Get all existing services to check ports
			services, err := client.ListServices("")
			if err != nil {
				fmt.Printf("❌ Failed to list services: %v\n", err)
				return
			}

			// Track used ports
			usedPorts := make(map[int]bool)
			for _, svc := range services {
				if svc.Spec.Type == "NodePort" {
					for _, port := range svc.Spec.Ports {
						if port.NodePort > 0 {
							usedPorts[port.NodePort] = true
						}
					}
				}
			}

			// Assign NodePorts if not specified
			for i := range service.Spec.Ports {
				port := &service.Spec.Ports[i]
				if port.NodePort == 0 {
					// Find available port in range 30000-32767
					port.NodePort = getAvailableNodePort(usedPorts)
					fmt.Printf("🔌 Assigned NodePort %d for port %d\n",
						port.NodePort, port.Port)
				} else if port.NodePort < 30000 || port.NodePort > 32767 {
					fmt.Printf("❌ Invalid NodePort %d: must be between 30000-32767\n",
						port.NodePort)
					return
				}
				usedPorts[port.NodePort] = true
			}
		}

		// Create service through API
		fmt.Printf("📦 Creating service '%s'...\n", service.Metadata.Name)
		if err := client.CreateService(service); err != nil {
			fmt.Printf("❌ Failed to create service: %v\n", err)
			return
		}

		fmt.Printf("✅ Service '%s' created successfully\n", service.Metadata.Name)

		// List matching pods
		pods, err := client.ListPods(service.Metadata.Namespace)
		if err != nil {
			fmt.Printf("⚠️ Failed to list pods: %v\n", err)
			return
		}

		matchCount := 0
		for _, pod := range pods {
			if matchLabels(pod.Metadata.Labels, service.Spec.Selector) {
				matchCount++
				//fmt.Printf("🔗 Pod '%s' matches service selector\n", pod.Metadata.Name)
			}
		}
		fmt.Printf("ℹ️ Found %d matching pods\n", matchCount)
	},
}

func init() {
	applyServiceCmd.Flags().StringVarP(&serviceFile, "file", "f", "", "YAML file to apply")
	applyServiceCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to apply the service to")
	applyServiceCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(applyServiceCmd)
}

// Helper function to get available NodePort
func getAvailableNodePort(usedPorts map[int]bool) int {
	rand.Seed(time.Now().UnixNano())
	for {
		port := rand.Intn(32767-30000+1) + 30000
		if !usedPorts[port] {
			return port
		}
	}
}

// Helper function to match labels
func matchLabels(podLabels, selector map[string]string) bool {
	for key, value := range selector {
		if podLabels[key] != value {
			return false
		}
	}
	return true
}
