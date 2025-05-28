package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/selimhanmrl/Own-Kubernetes/client"
	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var file string

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a YAML resource definition",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("❌ Error reading file: %v\n", err)
			return
		}

		// First, unmarshal to get the resource kind
		var resource struct {
			Kind string `yaml:"kind"`
		}
		if err := yaml.Unmarshal(data, &resource); err != nil {
			fmt.Printf("❌ Error parsing YAML: %v\n", err)
			return
		}

		// Create client
		c := client.NewClient(client.ClientConfig{
			Host: apiHost,
			Port: apiPort,
		})

		switch resource.Kind {
		case "Pod":
			var pod models.Pod
			if err := yaml.Unmarshal(data, &pod); err != nil {
				fmt.Printf("❌ Error parsing Pod YAML: %v\n", err)
				return
			}
			originalName := pod.Metadata.Name
			// Set initial pod status
			pod.Metadata.UID = uuid.New().String()
			pod.Status = models.PodStatus{
				Phase:     "Pending",
				StartTime: time.Now().Format(time.RFC3339),
			}
			pod.Metadata.Name = fmt.Sprintf("%s-%s-%s", originalName, pod.Metadata.UID[:4], pod.Metadata.UID[4:8])
			if err := c.CreatePod(pod); err != nil {
				fmt.Printf("❌ Error creating pod: %v\n", err)
				return
			}
			fmt.Printf("✅ Pod '%s' created successfully\n", pod.Metadata.Name)

			// Run scheduler immediately after pod creation
			//scheduler := exec.Command("go", "run", ".", "scheduler")
			//scheduler.Run()
		default:
			fmt.Printf("❌ Unsupported resource kind: %s\n", resource.Kind)
		}
	},
}

func init() {
	applyCmd.Flags().StringVarP(&file, "filename", "f", "", "YAML file containing the resource definition")
	applyCmd.MarkFlagRequired("filename")
}
