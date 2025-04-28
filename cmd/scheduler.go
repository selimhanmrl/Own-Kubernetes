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
        if namespace == "" {
            namespace = "default" // Default to 'default' namespace
        }

        // List all services in the namespace
        services := store.ListServices(namespace)
        if len(services) == 0 {
            fmt.Printf("No services found in namespace '%s'.\n", namespace)
            return
        }

        // List all pods in the namespace
        pods := store.ListPods(namespace)
        if len(pods) == 0 {
            fmt.Printf("No pods found in namespace '%s'.\n", namespace)
            return
        }

        // Iterate over services and match pods
        for _, service := range services {
            fmt.Printf("🔍 Processing service '%s'...\n", service.Name)

            // Find matching pods for the service
            matchingPods := []models.Pod{}
            for _, pod := range pods {
                matches := true
                for key, value := range service.Selector {
                    // Match service selector with pod labels
                    if pod.Metadata.Labels[key] != value {
                        matches = false
                        break
                    }
                }
                if matches {
                    matchingPods = append(matchingPods, pod)
                }
            }

            if len(matchingPods) == 0 {
                fmt.Printf("⚠️ No pods match the selector for service '%s'.\n", service.Name)
                continue
            }

            // Schedule matching pods
            for _, pod := range matchingPods {
                if pod.Status.Phase != "Pending" {
                    fmt.Printf("⚠️ Pod '%s' is already scheduled. Skipping.\n", pod.Metadata.Name)
                    continue
                }

                // Generate a unique container name
                containerName := fmt.Sprintf("%s-%s-%s", pod.Metadata.Name, pod.Spec.Containers[0].Name, pod.Metadata.UID[:8])

                // Extract resource constraints
                memoryLimit := pod.Spec.Containers[0].Resources.Limits["memory"]
                cpuLimit := pod.Spec.Containers[0].Resources.Limits["cpu"]

                // Convert memory limit from "Mi" to "m" if necessary
                if strings.HasSuffix(memoryLimit, "Mi") {
                    memoryLimit = strings.Replace(memoryLimit, "Mi", "m", 1)
                }
                if strings.HasSuffix(memoryLimit, "Gi") {
                    memoryLimit = strings.Replace(memoryLimit, "Gi", "g", 1)
                }
                if strings.HasSuffix(cpuLimit, "m") {
                    cpuLimit = strings.Replace(cpuLimit, "m", "", 1)
                    cpuLimitInt, err := strconv.Atoi(cpuLimit)
                    if err != nil {
                        fmt.Printf("❌ Failed to parse CPU limit for pod '%s': %v\n", pod.Metadata.Name, err)
                        continue
                    }
                    if cpuLimitInt > 1000 {
                        cpuLimitInt = cpuLimitInt / 1000
                    }
                    cpuLimit = fmt.Sprintf("%.2f", float64(cpuLimitInt)/1000) // Convert to core count
                }

                // Build the Docker run command with resource constraints and ports
                args := []string{"run", "-d", "--name", containerName}
                if memoryLimit != "" {
                    args = append(args, "--memory", memoryLimit)
                }
                if cpuLimit != "" {
                    args = append(args, "--cpus", cpuLimit)
                }

                // Add port mappings from the service
                for _, port := range service.Ports {
                    if service.Type == "NodePort" {
                        args = append(args, "-p", fmt.Sprintf("%d:%d", port.NodePort, port.TargetPort))
                    } else if service.Type == "ClusterIP" {
                        args = append(args, "-p", fmt.Sprintf("%d:%d", port.Port, port.TargetPort))
                    }
                }

                // Add the container image and command
                args = append(args, pod.Spec.Containers[0].Image)
                if len(pod.Spec.Containers[0].Cmd) > 0 {
                    args = append(args, pod.Spec.Containers[0].Cmd...)
                }

                // Start the Docker container
                out, err := exec.Command("docker", args...).Output()
                fmt.Printf("Running command: docker %s\n", strings.Join(args, " "))
                if err != nil {
                    fmt.Printf("❌ Failed to start container for pod '%s': %v\n", pod.Metadata.Name, err)
                    pod.Status.Phase = "Failed"
                } else {
                    pod.Status.Phase = "Running"
                    pod.Status.ContainerID = strings.TrimSpace(string(out))
                    fmt.Printf("✅ Scheduled and started pod '%s' for service '%s'.\n", pod.Metadata.Name, service.Name)
                }

                // Save the updated pod back to the store
                store.SavePod(pod)
            }
        }
    },
}

func init() {
    // Add namespace flag to the scheduler command
    schedulerCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace to filter services and pods")
    rootCmd.AddCommand(schedulerCmd)
}