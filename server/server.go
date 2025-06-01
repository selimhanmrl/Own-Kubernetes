package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/selimhanmrl/Own-Kubernetes/client"
	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/selimhanmrl/Own-Kubernetes/proxy"
	"github.com/selimhanmrl/Own-Kubernetes/store"
)

type APIServer struct {
	router       *mux.Router
	loadBalancer *proxy.LoadBalancer
	proxyServer  *proxy.ProxyServer
	client       *client.Client // HTTP client for making requests
}

func NewAPIServer() *APIServer {
	lb := proxy.NewLoadBalancer()
	server := &APIServer{
		router:       mux.NewRouter(),
		loadBalancer: lb,
		proxyServer:  proxy.NewProxyServer(lb),
		client:       client.NewClient(client.ClientConfig{Host: "localhost", Port: "8080"}),
	}

	// Add load balancer routes
	server.router.HandleFunc("/api/v1/loadbalancer/services", server.loadBalancer.RegisterService).Methods("POST")
	

	return server
}

func (s *APIServer) Start() {
	// Initialize components
	lb := proxy.NewLoadBalancer()
	proxyServer := proxy.NewProxyServer(lb)
	serviceUpdater := proxy.NewServiceUpdater(s.client, lb)

	// Start service updater
	serviceUpdater.Start()

	// Start proxy server on port 8080
	go http.ListenAndServe(":8080", proxyServer)

	s.setupRoutes()
	log.Printf("✅ API Server starting on port 8080")
	if err := http.ListenAndServe(":8080", s.router); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}

}

func (s *APIServer) setupRoutes() {
	// Pod endpoints
	fmt.Println("📝 Registering API routes...")

	s.router.HandleFunc("/api/v1/pods", s.handleListPods).Methods("GET")
	s.router.HandleFunc("/api/v1/namespaces/{namespace}/pods", s.handleListPodsByNamespace).Methods("GET")
	s.router.HandleFunc("/api/v1/pods", s.handleCreatePod).Methods("POST")
	s.router.HandleFunc("/api/v1/namespaces/{namespace}/pods/{name}", s.handleDeletePod).Methods("DELETE")
	s.router.HandleFunc("/api/v1/pods/{name}", s.handleDeletePod).Methods("DELETE")
	s.router.HandleFunc("/api/v1/pods/{name}", s.handleGetPod).Methods("GET")

	s.router.HandleFunc("/api/v1/namespaces/{namespace}/pods/{name}", s.handleUpdatePod).Methods("PUT")
	s.router.HandleFunc("/api/v1/pods/{name}/status", s.handleUpdatePodStatus).Methods("PUT")

	// Service endpoints
	s.router.HandleFunc("/api/v1/services", s.handleListServices).Methods("GET")
	s.router.HandleFunc("/api/v1/namespaces/{namespace}/services", s.handleListServicesByNamespace).Methods("GET")
	s.router.HandleFunc("/api/v1/services", s.handleCreateService).Methods("POST")

	// Node endpoints
	s.router.HandleFunc("/api/v1/nodes", s.handleListNodes).Methods("GET")
	s.router.HandleFunc("/api/v1/nodes", s.handleRegisterNode).Methods("POST")
	s.router.HandleFunc("/api/v1/nodes/{name}/status", s.handleUpdateNodeStatus).Methods("PUT")
}

func (s *APIServer) handleListPods(w http.ResponseWriter, r *http.Request) {
	nodeName := r.URL.Query().Get("fieldSelector")

	pods := store.ListAllPods()
	if nodeName != "" {
		// Filter pods by node name
		nodePods := []models.Pod{}
		for _, pod := range pods {
			if pod.Spec.NodeName == nodeName {
				nodePods = append(nodePods, pod)
			}
		}
		pods = nodePods
	}

	respondJSON(w, http.StatusOK, pods)
}

func (s *APIServer) handleListPodsByNamespace(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	pods := store.ListPods(namespace)
	respondJSON(w, http.StatusOK, pods)
}

func (s *APIServer) handleCreatePod(w http.ResponseWriter, r *http.Request) {
	var pod models.Pod
	if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := store.SavePod(pod); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, pod)
}

func (s *APIServer) handleDeletePod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	podName := vars["name"]

	fmt.Printf("🗑️ Handling delete request for pod: %s\n", podName)

	if err := store.DeletePod("default", podName); err != nil {
		fmt.Printf("❌ Failed to delete pod: %v\n", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fmt.Printf("✅ Successfully deleted pod: %s\n", podName)
	respondJSON(w, http.StatusOK, map[string]string{"message": "Pod deleted successfully"})
}

func (s *APIServer) handleUpdatePod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]

	var pod models.Pod
	if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate pod name and namespace match
	if pod.Metadata.Name != name || pod.Metadata.Namespace != namespace {
		respondError(w, http.StatusBadRequest, "Pod name/namespace mismatch")
		return
	}

	if err := store.SavePod(pod); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, pod)
}

func (s *APIServer) handleListServices(w http.ResponseWriter, r *http.Request) {
	services := store.ListServices("")
	respondJSON(w, http.StatusOK, services)
}

func (s *APIServer) handleListServicesByNamespace(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	services := store.ListServices(namespace)
	respondJSON(w, http.StatusOK, services)
}

func (s *APIServer) handleCreateService(w http.ResponseWriter, r *http.Request) {
	var service models.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := store.SaveService(service); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, service)
}

func (s *APIServer) handleListNodes(w http.ResponseWriter, r *http.Request) {
	nodes := store.ListNodes()
	respondJSON(w, http.StatusOK, nodes)
}

func (s *APIServer) handleRegisterNode(w http.ResponseWriter, r *http.Request) {
	var node models.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	fmt.Printf("🔌 Node '%s' attempting to connect from IP %s\n", node.Name, node.IP)

	if err := store.SaveNode(node); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fmt.Printf("✅ Node '%s' successfully registered and connected\n", node.Name)
	fmt.Printf("📊 Node details:\n")
	fmt.Printf("   - IP: %s\n", node.IP)
	fmt.Printf("   - Status: %s\n", node.Status.Phase)
	fmt.Printf("   - Capacity: CPU=%s, Memory=%s\n",
		node.Status.Capacity["cpu"],
		node.Status.Capacity["memory"])

	respondJSON(w, http.StatusCreated, node)
}

func (s *APIServer) handleGetPod(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	podName := vars["name"]

	pod, found := store.GetPod(podName)
	if !found {
		respondError(w, http.StatusNotFound, "Pod not found")
		return
	}

	respondJSON(w, http.StatusOK, pod)
}

func (s *APIServer) handleUpdateNodeStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	nodeName := vars["name"]

	var status models.NodeStatus
	if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	fmt.Printf("💓 Received heartbeat from node '%s'\n", nodeName)

	if err := store.UpdateNodeStatus(nodeName, status); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, status)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func (s *APIServer) handleUpdatePodStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	podName := vars["name"]
	fmt.Printf("🔍 Handling status update for pod: %s\n", podName)

	// Log request method and headers
	fmt.Printf("📝 Request Method: %s\n", r.Method)
	fmt.Printf("📝 Request Headers: %+v\n", r.Header)

	// Read and log the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read request body: %v\n", err)
		respondError(w, http.StatusBadRequest, "Failed to read request")
		return
	}
	fmt.Printf("📦 Request Body: %s\n", string(body))
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body)) // Reset body for later use

	var pod models.Pod
	if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
		fmt.Printf("❌ Failed to decode pod: %v\n", err)
		respondError(w, http.StatusBadRequest, "Invalid pod status payload")
		return
	}

	existingPod, found := store.GetPod(podName)
	fmt.Printf("🔍 Pod lookup result - Found: %v\n", found)
	if !found {
		fmt.Printf("❌ Pod not found: %s\n", podName)
		respondError(w, http.StatusNotFound, "Pod not found")
		return
	}

	fmt.Printf("📦 Existing pod status: %+v\n", existingPod.Status)
	fmt.Printf("📦 New pod status: %+v\n", pod.Status)

	// Update status fields
	existingPod.Status = pod.Status
	if err := store.SavePod(existingPod); err != nil {
		fmt.Printf("❌ Failed to save pod: %v\n", err)
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	fmt.Printf("✅ Successfully updated pod status\n")
	respondJSON(w, http.StatusOK, existingPod)
}
