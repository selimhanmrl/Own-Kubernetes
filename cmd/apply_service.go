package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
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
			if namespace != "" {
				service.Metadata.Namespace = namespace
			} else {
				service.Metadata.Namespace = "default"
			}
		}

		fmt.Printf("Selector: %v\n", service.Spec.Selector)
		// Validate selector
		if len(service.Spec.Selector) == 0 {
			fmt.Println("❌ Service must have a selector")
			return
		}
		// Modify the NodePort assignment section in applyServiceCmd
		// Replace the NodePort section with this updated version:
		if service.Spec.Type == "NodePort" {
			usedPorts := getUsedNodePorts()
			replicaPorts := make(map[int][]int)

			// First, validate ports and assign base NodePorts
			for i := range service.Spec.Ports {
				port := &service.Spec.Ports[i]
				if port.NodePort == 0 {
					// Generate base NodePort for the service
					baseNodePort := getRandomNodePort(usedPorts)
					port.NodePort = baseNodePort
					usedPorts[baseNodePort] = true

					// Initialize empty ports array for this service port
					replicaPorts[port.Port] = []int{}
				}
			}

			// Initialize annotations if nil
			if service.Metadata.Annotations == nil {
				service.Metadata.Annotations = make(map[string]string)
			}

			// Get matching pods
			pods := store.ListPods(service.Metadata.Namespace)
			matchingPods := []models.Pod{}
			for _, pod := range pods {
				matches := true
				for key, value := range service.Spec.Selector {
					if pod.Metadata.Labels[key] != value {
						matches = false
						break
					}
				}
				if matches {
					matchingPods = append(matchingPods, pod)
				}
			}

			fmt.Printf("Found %d matching pods\n", len(matchingPods))

			// Generate ports for each service port
			for i := range service.Spec.Ports {
				port := &service.Spec.Ports[i]

				if len(matchingPods) > 0 {
					// Generate unique ports for each replica
					ports := make([]int, len(matchingPods))
					for j := range matchingPods {
						newPort := getRandomNodePort(usedPorts)
						ports[j] = newPort
						usedPorts[newPort] = true
					}

					// Store ports for this service port
					replicaPorts[port.Port] = ports

					fmt.Printf("ℹ️ Assigned NodePorts for port %d:\n", port.Port)
					for j, replicaPort := range ports {
						fmt.Printf("  - %s: %d\n", matchingPods[j].Metadata.Name, replicaPort)
					}
				} else {
					fmt.Printf("⚠️ No matching pods found for service port %d\n", port.Port)
					replicaPorts[port.Port] = []int{port.NodePort}
				}

				// Store in annotations
				portsStr := fmt.Sprintf("%v", replicaPorts[port.Port])
				portsStr = strings.Trim(portsStr, "[]")
				service.Metadata.Annotations[fmt.Sprintf("nodeports.%d", port.Port)] = fmt.Sprintf("[%s]", portsStr)
			}

			fmt.Printf("ℹ️ Storing NodePorts in service annotations:\n")
			for port, ports := range replicaPorts {
				fmt.Printf("Port %d -> %v\n", port, ports)
			}
		}

		// Check for matching pods
		pods := store.ListPods(service.Metadata.Namespace)
		matchingPods := 0
		for _, pod := range pods {
			matches := true
			for key, value := range service.Spec.Selector {
				if pod.Metadata.Labels[key] != value {
					matches = false
					break
				}
			}
			if matches {
				matchingPods++
			}
			fmt.Println("Pod:", pod.Metadata.Name, "Matches:", matches)
			fmt.Println("Service Annotations:", service.Metadata.Annotations)
		}

		fmt.Printf("ℹ️ Found %d matching pods for service '%s'\n",
			matchingPods, service.Metadata.Name)

		// Save the service
		if err := store.SaveService(service); err != nil {
			fmt.Printf("❌ Failed to save service: %v\n", err)
			return
		}

		fmt.Printf("✅ Service '%s' created successfully in namespace '%s'\n",
			service.Metadata.Name, service.Metadata.Namespace)
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
		for _, port := range service.Spec.Ports {
			if service.Spec.Type == "NodePort" && port.NodePort > 0 {
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

func generateReplicaPorts(baseNodePort int, replicas int, usedPorts map[int]bool) []int {
	ports := make([]int, replicas)

	// Generate unique ports for all replicas
	for i := 0; i < replicas; i++ {
		port := getRandomNodePort(usedPorts)
		ports[i] = port
		usedPorts[port] = true
	}
	return ports
}
