package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/selimhanmrl/Own-Kubernetes/store"
	"github.com/spf13/cobra"
)

var getServicesCmd = &cobra.Command{
	Use:   "get services",
	Short: "Get a list of services in a namespace or all namespaces",
	Run: func(cmd *cobra.Command, args []string) {
		var services []models.Service

		if allNamespaces {
			services = store.ListServices("") // List all services across all namespaces
		} else {
			if namespace == "" {
				namespace = "default" // Default to 'default' namespace
			}
			services = store.ListServices(namespace)
		}

		if len(services) == 0 {
			if allNamespaces {
				fmt.Println("No services found in any namespace.")
			} else {
				fmt.Printf("No services found in namespace '%s'.\n", namespace)
			}
			return
		}

		// Create a tabular writer for output
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if allNamespaces {
			fmt.Fprintln(w, "NAMESPACE\tNAME\tTYPE\tPORTS")
		} else {
			fmt.Fprintln(w, "NAME\tTYPE\tPORTS")
		}

		for _, service := range services {
			ports := ""
			for _, port := range service.Ports {
				ports += fmt.Sprintf("%d:%d ", port.Port, port.TargetPort)
			}

			if allNamespaces {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					service.Namespace,
					service.Name,
					service.Type,
					ports,
				)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					service.Name,
					service.Type,
					ports,
				)
			}
		}

		w.Flush()
	},
}

func init() {
	getServicesCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace to filter services")
	getServicesCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "List services across all namespaces")
	rootCmd.AddCommand(getServicesCmd)
}
