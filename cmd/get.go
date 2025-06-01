package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/selimhanmrl/Own-Kubernetes/client"
	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/spf13/cobra"
)

var namespace string   // Namespace flag
var allNamespaces bool // Add -A flag

var getCmd = &cobra.Command{
	Use:   "get pods",
	Short: "Get a list of pods in a namespace or all namespaces",
	Run: func(cmd *cobra.Command, args []string) {
		c := client.NewClient(client.ClientConfig{
			Host: apiHost,
			Port: apiPort,
		})

		var pods []models.Pod
		var err error

		if allNamespaces {
			pods, err = c.ListPods("")
		} else {
			if namespace == "" {
				namespace = "default"
			}
			pods, err = c.ListPods(namespace)
		}

		if err != nil {
			fmt.Printf("Failed to list pods: %v\n", err)
			return
		}

		if len(pods) == 0 {
			if allNamespaces {
				fmt.Println("No pods found in any namespace.")
			} else {
				fmt.Printf("No pods found in namespace '%s'.\n", namespace)
			}
			return
		}

		// Sort pods by namespace if listing all namespaces
		if allNamespaces {
			sort.Slice(pods, func(i, j int) bool {
				return pods[i].Metadata.Namespace < pods[j].Metadata.Namespace
			})
		}

		// Create a tabular writer for output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if allNamespaces {
			fmt.Fprintln(w, "NAMESPACE\tNAME\tREADY\tSTATUS\tRESTARTS\tAGE\tNODEPORT\tRESOURCES")
		} else {
			fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE\tNODEPORT\tRESOURCES")
		}

		for _, pod := range pods {
			ready := fmt.Sprintf("%d/%d", len(pod.Spec.Containers), len(pod.Spec.Containers))
			restarts := "0"

			age := "unknown"
			if pod.Status.StartTime != "" {
				if t, err := time.Parse(time.RFC3339, pod.Status.StartTime); err == nil {
					duration := time.Since(t).Round(time.Second)
					age = duration.String()
				}
			}

			// Get NodePort if assigned
			nodePort := "-"
			if port, exists := c.GetAssignedNodePort(pod.Metadata.Name); exists {
				nodePort = fmt.Sprintf("%d", port)
			}

			resourceInfo := ""
			for _, container := range pod.Spec.Containers {
				resourceInfo += fmt.Sprintf("[%s: Requests(cpu=%s, mem=%s), Limits(cpu=%s, mem=%s)] ",
					container.Name,
					container.Resources.Requests["cpu"],
					container.Resources.Requests["memory"],
					container.Resources.Limits["cpu"],
					container.Resources.Limits["memory"],
				)
			}

			if allNamespaces {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					pod.Metadata.Namespace,
					pod.Metadata.Name,
					ready,
					pod.Status.Phase,
					restarts,
					age,
					nodePort,
					resourceInfo,
				)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					pod.Metadata.Name,
					ready,
					pod.Status.Phase,
					restarts,
					age,
					nodePort,
					resourceInfo,
				)
			}
		}

		w.Flush()
	},
}

func init() {
	// Add namespace and all-namespaces flags to the get command
	getCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace to filter pods")
	getCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List pods across all namespaces")
	rootCmd.AddCommand(getCmd)
}
