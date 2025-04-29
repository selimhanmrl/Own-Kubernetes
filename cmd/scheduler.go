package cmd

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/selimhanmrl/Own-Kubernetes/store"
	"github.com/spf13/cobra"
)

var schedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Run the scheduler to assign pods to nodes and configure services",
	Run: func(cmd *cobra.Command, args []string) {
		// Get all pods from all namespaces if no specific namespace is provided
		var pods []models.Pod
		if namespace != "" {
			pods = store.ListPods(namespace)
		} else {
			pods = store.ListPods("")
		}

		if len(pods) == 0 {
			fmt.Println("No pods found to schedule.")
			return
		}

		// Schedule all pending pods
		for _, pod := range pods {
			if pod.Status.Phase != "Pending" {
				fmt.Printf("⚠️ Pod '%s' in namespace '%s' is already scheduled. Skipping.\n",
					pod.Metadata.Name, pod.Metadata.Namespace)
				continue
			}

			fmt.Printf("📦 Scheduling pod '%s' in namespace '%s'...\n",
				pod.Metadata.Name, pod.Metadata.Namespace)

			// Generate a unique container name
			containerName := fmt.Sprintf("%s-%s-%s",
				pod.Metadata.Name, pod.Spec.Containers[0].Name, pod.Metadata.UID[:8])

			// Extract and convert resource constraints
			memoryLimit := pod.Spec.Containers[0].Resources.Limits["memory"]
			cpuLimit := pod.Spec.Containers[0].Resources.Limits["cpu"]

			// Convert memory limit from "Mi" to "m" if necessary
			if strings.HasSuffix(memoryLimit, "Mi") {
				memoryLimit = strings.Replace(memoryLimit, "Mi", "m", 1)
			}
			if strings.HasSuffix(memoryLimit, "Gi") {
				memoryLimit = strings.Replace(memoryLimit, "Gi", "g", 1)
			}

			// Convert CPU limit
			if strings.HasSuffix(cpuLimit, "m") {
				cpuLimit = strings.Replace(cpuLimit, "m", "", 1)
				cpuLimitInt, err := strconv.Atoi(cpuLimit)
				if err != nil {
					fmt.Printf("❌ Failed to parse CPU limit for pod '%s': %v\n",
						pod.Metadata.Name, err)
					continue
				}
				if cpuLimitInt > 1000 {
					cpuLimitInt = cpuLimitInt / 1000
				}
				cpuLimit = fmt.Sprintf("%.2f", float64(cpuLimitInt)/1000)
			}

			// Build the Docker run command with resource constraints
			args := []string{"run", "-d", "--name", containerName}
			if memoryLimit != "" {
				args = append(args, "--memory", memoryLimit)
			}
			if cpuLimit != "" {
				args = append(args, "--cpus", cpuLimit)
			}

			// Look for matching services in the pod's namespace
			services := store.ListServices(pod.Metadata.Namespace)
			for _, service := range services {
				// Skip services from different namespaces
				if service.Metadata.Namespace != pod.Metadata.Namespace {
					continue
				}

				// Check if pod labels match service selector
				matches := true
				for key, value := range service.Spec.Selector {
					if pod.Metadata.Labels[key] != value {
						matches = false
						break
					}
				}

				if matches {
					fmt.Printf("🔗 Pod matches service '%s' in namespace '%s'\n",
						service.Metadata.Name, service.Metadata.Namespace)

					for _, port := range service.Spec.Ports {
						if service.Spec.Type == "NodePort" {
							usedPorts := getUsedPortsFromService(service)

							// Always generate a new port for each pod
							newPort := getRandomNodePort(usedPorts)
							args = append(args, "-p", fmt.Sprintf("%d:%d", newPort, port.TargetPort))
							fmt.Printf("📌 Assigned new NodePort %d for pod %s\n", newPort, pod.Metadata.Name)

							// Update service annotations
							annotationKey := fmt.Sprintf("nodeports.%d", port.Port)
							portList := []string{}

							// Parse existing ports
							if portListStr, ok := service.Metadata.Annotations[annotationKey]; ok {
								portListStr = strings.Trim(portListStr, "[]")
								if portListStr != "" {
									portList = strings.Split(portListStr, ",")
								}
							}

							// Add new port
							portList = append(portList, fmt.Sprintf("%d", newPort))

							// Update annotation
							service.Metadata.Annotations[annotationKey] = fmt.Sprintf("[%s]", strings.Join(portList, ","))
							store.SaveService(service)
						}
					}
				}
			}

			// Add container image and command
			args = append(args, pod.Spec.Containers[0].Image)
			if len(pod.Spec.Containers[0].Cmd) > 0 {
				args = append(args, pod.Spec.Containers[0].Cmd...)
			}

			// Start the container
			fmt.Printf("Running command: docker %s\n", strings.Join(args, " "))
			out, err := exec.Command("docker", args...).Output()
			if err != nil {
				fmt.Printf("❌ Failed to start container for pod '%s': %v\n",
					pod.Metadata.Name, err)
				pod.Status.Phase = "Failed"
			} else {
				pod.Status.Phase = "Running"
				pod.Status.ContainerID = strings.TrimSpace(string(out))
				fmt.Printf("✅ Successfully scheduled pod '%s' in namespace '%s'\n",
					pod.Metadata.Name, pod.Metadata.Namespace)
			}

			// Save the updated pod back to the store
			store.SavePod(pod)
		}
	},
}

func init() {
	schedulerCmd.Flags().StringVarP(&namespace, "namespace", "n", "",
		"Namespace to filter services and pods (optional)")
	rootCmd.AddCommand(schedulerCmd)
}

// Add this helper function at the top of the file
func getUsedPortsFromService(service models.Service) map[int]bool {
	usedPorts := make(map[int]bool)
	for _, port := range service.Spec.Ports {
		if portListStr, ok := service.Metadata.Annotations[fmt.Sprintf("nodeports.%d", port.Port)]; ok {
			portListStr = strings.Trim(portListStr, "[]")
			for _, p := range strings.Split(portListStr, ",") {
				if nodePort, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
					usedPorts[nodePort] = true
				}
			}
		}
	}
	return usedPorts
}
