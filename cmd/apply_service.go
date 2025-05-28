package cmd

import (
	"fmt"
	"os"

	"github.com/selimhanmrl/Own-Kubernetes/client"
	"github.com/selimhanmrl/Own-Kubernetes/models"
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

		// Validate selector
		if len(service.Spec.Selector) == 0 {
			fmt.Println("❌ Service must have a selector")
			return
		}

		// Create client
		c := client.NewClient(client.ClientConfig{
			Host: apiHost,
			Port: apiPort,
		})

		// Create the service through API
		if err := c.CreateService(service); err != nil {
			fmt.Printf("❌ Failed to create service: %v\n", err)
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

