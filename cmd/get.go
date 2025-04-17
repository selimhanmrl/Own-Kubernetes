package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/selimhanmrl/Own-Kubernetes/store"
	"github.com/spf13/cobra"
)

var namespace string   // Namespace flag
var allNamespaces bool // Add -A flag

var getCmd = &cobra.Command{
	Use:   "get pods",
	Short: "Get a list of pods in a namespace or all namespaces",
	Run: func(cmd *cobra.Command, args []string) {
		var pods []models.Pod

		if allNamespaces {
			// List all pods across all namespaces
			pods = store.ListAllPods()

			// Sort pods by namespace
			sort.Slice(pods, func(i, j int) bool {
				return pods[i].Metadata.Namespace < pods[j].Metadata.Namespace
			})
		} else {
			// Default to 'default' namespace if no namespace is provided
			if namespace == "" {
				namespace = "default"
			}
			// List pods in the specified namespace
			pods = store.ListPods(namespace)
		}

		if len(pods) == 0 {
			if allNamespaces {
				fmt.Println("No pods found in any namespace.")
			} else {
				fmt.Printf("No pods found in namespace '%s'.\n", namespace)
			}
			return
		}

		// Create a tabular writer for output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAMESPACE\tNAME\tREADY\tSTATUS\tRESTARTS\tAGE")

		for _, pod := range pods {
			ready := fmt.Sprintf("%d/%d", len(pod.Spec.Containers), len(pod.Spec.Containers)) // Assume all containers are ready
			restarts := "0"                                                                   // Placeholder for restarts (not implemented yet)

			age := "unknown"
			if pod.Status.StartTime != "" {
				if t, err := time.Parse(time.RFC3339, pod.Status.StartTime); err == nil {
					duration := time.Since(t).Round(time.Second)
					age = duration.String()
				}
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				pod.Metadata.Namespace,
				pod.Metadata.Name,
				ready,
				pod.Status.Phase,
				restarts,
				age,
			)

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
