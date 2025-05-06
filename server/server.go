package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/selimhanmrl/Own-Kubernetes/models"
	"github.com/selimhanmrl/Own-Kubernetes/store"
)

type APIServer struct {
	router *mux.Router
}

func NewAPIServer() *APIServer {
	return &APIServer{
		router: mux.NewRouter(),
	}
}

func (s *APIServer) Start() {
	s.setupRoutes()
	log.Printf("✅ API Server starting on port 8080")
	if err := http.ListenAndServe(":8080", s.router); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}

func (s *APIServer) setupRoutes() {
	// Pod endpoints
	s.router.HandleFunc("/api/v1/pods", s.handleListPods).Methods("GET")
	s.router.HandleFunc("/api/v1/namespaces/{namespace}/pods", s.handleListPodsByNamespace).Methods("GET")
	s.router.HandleFunc("/api/v1/pods", s.handleCreatePod).Methods("POST")
	s.router.HandleFunc("/api/v1/namespaces/{namespace}/pods/{name}", s.handleDeletePod).Methods("DELETE")

	// Service endpoints
	s.router.HandleFunc("/api/v1/services", s.handleListServices).Methods("GET")
	s.router.HandleFunc("/api/v1/namespaces/{namespace}/services", s.handleListServicesByNamespace).Methods("GET")
	s.router.HandleFunc("/api/v1/services", s.handleCreateService).Methods("POST")
}

func (s *APIServer) handleListPods(w http.ResponseWriter, r *http.Request) {
	pods := store.ListAllPods()
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
	namespace := vars["namespace"]
	name := vars["name"]

	if success := store.DeletePodByName(name, namespace); !success {
		respondError(w, http.StatusNotFound, "Pod not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Pod deleted successfully"})
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

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
