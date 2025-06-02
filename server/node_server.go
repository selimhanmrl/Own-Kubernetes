package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/selimhanmrl/Own-Kubernetes/agent"
	"github.com/selimhanmrl/Own-Kubernetes/models"
)

type NodeServer struct {
	router *mux.Router
	name   string
	port   string
	nodeIP string // Add nodeIP field
	agent  *agent.NodeAgent
}

func NewNodeServer(name, port, nodeIP, apiHost, apiPort string) *NodeServer {
	return &NodeServer{
		router: mux.NewRouter(),
		name:   name,
		port:   port,
		nodeIP: nodeIP,
		agent:  agent.NewNodeAgent(name, nodeIP, apiHost, apiPort),
	}
}

func (s *NodeServer) Start() error {
	// Validate connection to API server first
	apiAddr := fmt.Sprintf("http://%s:%s", s.agent.GetClient().GetConfig().Host, s.agent.GetClient().GetConfig().Port)
	fmt.Printf("📡 Checking API server connection at %s...\n", apiAddr)

	// Try to connect to API server with timeout
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	_, err := client.Get(fmt.Sprintf("%s/healthz", apiAddr))
	if err != nil {
		return fmt.Errorf("❌ cannot connect to API server at %s: %v\n"+
			apiAddr, err)
	}

	fmt.Printf("✅ Successfully connected to API server\n")

	// Start the node agent
	if err := s.agent.Start(); err != nil {
		return fmt.Errorf("failed to start node agent: %v", err)
	}

	s.setupRoutes()

	// Start pod watcher in background
	go s.watchForPods()

	// Bind to specific IP and port
	addr := fmt.Sprintf("%s:%s", s.nodeIP, s.port)
	fmt.Printf("🚀 Starting node server %s on %s\n", s.name, addr)
	return http.ListenAndServe(addr, s.router)
}

func (s *NodeServer) setupRoutes() {
	s.router.HandleFunc("/healthz", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/pods", s.handleListPods).Methods("GET")
	s.router.HandleFunc("/pods", s.handleCreatePod).Methods("POST")
	s.router.HandleFunc("/pods/{name}", s.handleDeletePod).Methods("DELETE")
	s.router.HandleFunc("/pods/{name}/status", s.handleUpdatePodStatus).Methods("PUT")
	s.router.HandleFunc("/metrics", s.handleMetrics).Methods("GET")
}

func (s *NodeServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Node %s is healthy", s.name)
}

func (s *NodeServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// TODO: Add node metrics collection
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Node %s metrics", s.name)
}

func (s *NodeServer) handleListPods(w http.ResponseWriter, r *http.Request) {
	pods, err := s.agent.ListPods()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, pods)
}

func (s *NodeServer) handleCreatePod(w http.ResponseWriter, r *http.Request) {
	var pod models.Pod
	if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := s.agent.StartPod(&pod); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, pod)
}

func (s *NodeServer) handleDeletePod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	podName := vars["name"]

	if err := s.agent.CleanupPod(podName); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Pod deleted successfully"})
}

func (s *NodeServer) handleUpdatePodStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	podName := vars["name"]

	var status models.PodStatus
	if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Check if container exists
	cmd := exec.Command("docker", "inspect", podName)
	if err := cmd.Run(); err != nil {
		respondError(w, http.StatusNotFound, fmt.Sprintf("Pod %s not found", podName))
		return
	}

	// Use nodeIP instead of name for HostIP
	status.HostIP = s.nodeIP // Fix: Use nodeIP instead of name
	respondJSON(w, http.StatusOK, status)
}

func (s *NodeServer) watchForPods() {
	fmt.Printf("👀 Starting pod watcher for node %s\n", s.name)
	ticker := time.NewTicker(5 * time.Second)
	previousPods := make(map[string]bool)

	for range ticker.C {
		currentPods := make(map[string]bool)

		// Get assigned pods from API server
		client := s.agent.GetClient()
		pods, err := client.ListPods(fmt.Sprintf("?fieldSelector=spec.nodeName=%s", s.name))
		if err != nil {
			//fmt.Printf("❌ Failed to list pods: %v\n", err)
			continue
		}

		// Process each pod
		for _, pod := range pods {
			currentPods[pod.Metadata.Name] = true

			// Always update pod's HostIP
			pod.Status.HostIP = s.nodeIP

			fmt.Printf("📦 Found pod %s (status: %s)\n",
				pod.Metadata.Name, pod.Status.Phase)

			switch pod.Status.Phase {
			case "Pending":
				fmt.Printf("🚀 Starting pod %s on node %s\n",
					pod.Metadata.Name, s.name)

				if err := s.agent.StartPod(&pod); err != nil {
					fmt.Printf("❌ Failed to start pod %s: %v\n",
						pod.Metadata.Name, err)
					// Update pod status to Failed
					pod.Status.Phase = "Failed"
					if err := client.UpdatePodStatus(&pod); err != nil {
						fmt.Printf("❌ Failed to update pod status: %v\n", err)
					}
					continue
				}
				fmt.Printf("✅ Successfully started pod %s\n", pod.Metadata.Name)

			case "Running":
				// Check if container is actually running
				cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", pod.Metadata.Name)
				output, err := cmd.CombinedOutput()
				if err != nil || strings.TrimSpace(string(output)) != "true" {
					fmt.Printf("⚠️ Pod %s marked as Running but container is not running\n",
						pod.Metadata.Name)
					pod.Status.Phase = "Failed"
					if err := client.UpdatePodStatus(&pod); err != nil {
						fmt.Printf("❌ Failed to update pod status: %v\n", err)
					}
				}
			}
		}

		// Check for deleted pods
		for podName := range previousPods {
			if !currentPods[podName] {
				fmt.Printf("🗑️ Pod %s was deleted, cleaning up\n", podName)
				if err := s.agent.CleanupPod(podName); err != nil {
					fmt.Printf("❌ Failed to cleanup pod %s: %v\n", podName, err)
				}
			}
		}

		previousPods = currentPods
	}
}
