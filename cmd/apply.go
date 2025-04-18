package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/selimhanmrl/Own-Kubernetes/store"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var file string

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a YAML pod definition",
	Run: func(cmd *cobra.Command, args []string) {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Println("❌ Failed to read file:", err)
			return
		}

		var pod models.Pod
		if err := yaml.Unmarshal(data, &pod); err != nil {
			fmt.Println("❌ Failed to parse YAML:", err)
			return
		}

		// Set namespace if not provided in the YAML
		if pod.Metadata.Namespace == "" {
			if namespace == "" {
				namespace = "default" // Default to 'default' namespace
			}
			pod.Metadata.Namespace = namespace
		}

		// Check if a pod with the same name already exists
		originalName := pod.Metadata.Name

		pod.Metadata.UID = uuid.NewString()
		pod.Status.Phase = "Pending"
		pod.Status.StartTime = time.Now().Format("2006-01-02T15:04:05Z07:00")
		pod.Metadata.Name = fmt.Sprintf("%s-%s-%s", originalName, pod.Metadata.UID[:4], pod.Metadata.UID[4:8])
		store.SavePod(pod)

		out, _ := json.MarshalIndent(pod, "", "  ")
		fmt.Println("✅ Pod created:")
		fmt.Println(string(out))
	},
}

func init() {
	applyCmd.Flags().StringVarP(&file, "file", "f", "", "YAML file to apply")
	applyCmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to apply the pod to")
	applyCmd.MarkFlagRequired("file")
	rootCmd.AddCommand(applyCmd)
}
