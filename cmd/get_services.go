package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/selimhanmrl/Own-Kubernetes/client"
	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/spf13/cobra"
)

var getServicesCmd = &cobra.Command{
	Use:   "get services",
	Short: "Get a list of services in a namespace or all namespaces",
	Run: func(cmd *cobra.Command, args []string) {
		c := client.NewClient(client.ClientConfig{
			Host: apiHost,
			Port: apiPort,
		})

		var services []models.Service
		var err error

		if allNamespaces {
			services, err = c.ListServices("")
		} else {
			if namespace == "" {
				namespace = "default"
			}
			services, err = c.ListServices(namespace)
		}

		if err != nil {
			fmt.Printf("Failed to list services: %v\n", err)
			return
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
			for _, port := range service.Spec.Ports {
				ports += fmt.Sprintf("%d:%d ", port.Port, port.TargetPort)
			}

			if allNamespaces {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					service.Metadata.Namespace,
					service.Metadata.Name,
					service.Spec.Type,
					ports,
				)
			} else {
				fmt.Fprintf(w, "%s\t%s\t%s\n",
					service.Metadata.Name,
					service.Spec.Type,
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
