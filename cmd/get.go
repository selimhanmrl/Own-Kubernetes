package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/selimhanmrl/Own-Kubernetes/store"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get resources",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 || args[0] != "pods" {
			fmt.Println("❌ Only 'get pods' is supported right now")
			return
		}

		pods := store.ListPods()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE")

		for _, pod := range pods {
			ready := fmt.Sprintf("%d/%d", len(pod.Spec.Containers), len(pod.Spec.Containers)) // hepsi çalışıyor varsay
			restarts := "0"

			age := "unknown"
			if pod.Status.StartTime != "" {
				if t, err := time.Parse(time.RFC3339, pod.Status.StartTime); err == nil {
					duration := time.Since(t).Round(time.Second)
					age = duration.String()
				}
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
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
