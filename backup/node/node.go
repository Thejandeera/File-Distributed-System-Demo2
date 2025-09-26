package node

import (
	"distributedfs/config"
	"distributedfs/consensus"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Node represents a distributed file system node with Raft consensus
type Node struct {
	Port        string
	StoragePath string
	RaftAddr    string
	RaftDir     string
	NodeID      string
	Server      *http.Server
	Consensus   *consensus.RaftConsensus
	isBootstrap bool
}

// NewNode creates a new node instance with Raft consensus
func NewNode(port, nodeID string, isBootstrap bool) (*Node, error) {
	// Initialize configuration
	config.InitializeConfig()

	storagePath := filepath.Join(config.GetStoragePath(), "node_"+port)
	raftDir := filepath.Join(storagePath, "raft")
	filesDir := filepath.Join(storagePath, "files")
	raftAddr := fmt.Sprintf("localhost:%d", getPortFromString(port)+1000) // Raft on different port

	// Create directories
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	// Initialize Raft consensus
	raftConsensus, err := consensus.NewRaftConsensus(nodeID, raftAddr, raftDir, filesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Raft consensus: %v", err)
	}

	node := &Node{
		Port:        port,
		StoragePath: storagePath,
		RaftAddr:    raftAddr,
		RaftDir:     raftDir,
		NodeID:      nodeID,
		Consensus:   raftConsensus,
		isBootstrap: isBootstrap,
	}

	return node, nil
}

// Start initializes and starts the node
func (n *Node) Start() error {
	// Bootstrap or join cluster
	if n.isBootstrap {
		log.Printf("Bootstrapping new cluster as node %s", n.NodeID)
		if err := n.Consensus.Bootstrap(); err != nil {
			return fmt.Errorf("failed to bootstrap cluster: %v", err)
		}
	} else {
		// Wait a moment for bootstrap node to be ready
		time.Sleep(2 * time.Second)
		log.Printf("Attempting to join existing cluster as node %s", n.NodeID)
		// In a real implementation, you'd discover the leader automatically
		// For now, we'll assume the bootstrap node is on port 8000
		if err := n.Consensus.Join(n.NodeID, n.RaftAddr); err != nil {
			log.Printf("Failed to join cluster: %v", err)
			// Continue anyway, node might still work
		}
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	n.setupRoutes(mux)

	// Create HTTP server
	n.Server = &http.Server{
		Addr:    ":" + n.Port,
		Handler: n.enableCORS(mux),
	}

	log.Printf("🟢 Node %s running on port %s (Raft: %s)", n.NodeID, n.Port, n.RaftAddr)
	return n.Server.ListenAndServe()
}

// setupRoutes configures HTTP routes
func (n *Node) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/upload", n.uploadHandler)
	mux.HandleFunc("/download", n.downloadHandler)
	mux.HandleFunc("/files", n.filesHandler)
	mux.HandleFunc("/delete", n.deleteHandler)
	mux.HandleFunc("/health", n.healthHandler)
	mux.HandleFunc("/stats", n.statsHandler)
	mux.HandleFunc("/raft/stats", n.raftStatsHandler)
	mux.HandleFunc("/raft/leader", n.raftLeaderHandler)
	mux.HandleFunc("/raft/join", n.raftJoinHandler)
}

// enableCORS middleware
func (n *Node) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE, PUT")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			return
		}

		next.ServeHTTP(w, r)
	})
}

// uploadHandler handles file uploads through Raft consensus
func (n *Node) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if this node is the leader
	if !n.Consensus.IsLeader() {
		leader := n.Consensus.GetLeader()
		if leader != "" {
			// Redirect to leader
			leaderURL := fmt.Sprintf("http://%s:%s/upload",
				strings.Split(leader, ":")[0],
				getHTTPPortFromRaftAddr(leader))
			http.Redirect(w, r, leaderURL, http.StatusTemporaryRedirect)
			return
		}
		http.Error(w, "No leader available", http.StatusServiceUnavailable)
		return
	}

	// Parse multipart form
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to read file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file content: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply command through Raft
	if err := n.Consensus.ApplyCommand("upload", header.Filename, fileBytes); err != nil {
		http.Error(w, "Failed to replicate file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ File uploaded via Raft: %s", header.Filename)
	fmt.Fprintf(w, "✅ File uploaded successfully: %s", header.Filename)
}

// deleteHandler handles file deletion through Raft consensus
func (n *Node) deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" && r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := r.URL.Query().Get("name")
	if filename == "" {
		http.Error(w, "Missing filename parameter", http.StatusBadRequest)
		return
	}

	// Check if this node is the leader
	if !n.Consensus.IsLeader() {
		leader := n.Consensus.GetLeader()
		if leader != "" {
			// Redirect to leader
			leaderURL := fmt.Sprintf("http://%s:%s/delete?name=%s",
				strings.Split(leader, ":")[0],
				getHTTPPortFromRaftAddr(leader),
				filename)
			http.Redirect(w, r, leaderURL, http.StatusTemporaryRedirect)
			return
		}
		http.Error(w, "No leader available", http.StatusServiceUnavailable)
		return
	}

	// Apply command through Raft
	if err := n.Consensus.ApplyCommand("delete", filename, nil); err != nil {
		http.Error(w, "Failed to delete file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("✅ File deleted via Raft: %s", filename)
	fmt.Fprintf(w, "✅ File deleted successfully: %s", filename)
}

// downloadHandler serves files directly (read-only operation)
func (n *Node) downloadHandler(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("name")
	if filename == "" {
		http.Error(w, "Missing filename parameter", http.StatusBadRequest)
		return
	}

	filesDir := filepath.Join(n.StoragePath, "files")
	filePath := filepath.Join(filesDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, filePath)
}

// filesHandler returns list of files
func (n *Node) filesHandler(w http.ResponseWriter, r *http.Request) {
	filesDir := filepath.Join(n.StoragePath, "files")
	files, err := os.ReadDir(filesDir)
	if err != nil {
		http.Error(w, "Failed to read directory", http.StatusInternalServerError)
		return
	}

	var fileNames []string
	for _, file := range files {
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileNames)
}

// healthHandler returns node health
func (n *Node) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// statsHandler returns node statistics
func (n *Node) statsHandler(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"nodeId":    n.NodeID,
		"port":      n.Port,
		"raftAddr":  n.RaftAddr,
		"isLeader":  n.Consensus.IsLeader(),
		"leader":    n.Consensus.GetLeader(),
		"raftState": n.Consensus.GetState(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// raftStatsHandler returns detailed Raft statistics
func (n *Node) raftStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats := n.Consensus.GetStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// raftLeaderHandler returns current leader information
func (n *Node) raftLeaderHandler(w http.ResponseWriter, r *http.Request) {
	leader := map[string]interface{}{
		"leader":   n.Consensus.GetLeader(),
		"isLeader": n.Consensus.IsLeader(),
		"state":    n.Consensus.GetState(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leader)
}

// raftJoinHandler allows nodes to join the cluster
func (n *Node) raftJoinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var joinRequest struct {
		NodeID   string `json:"nodeId"`
		RaftAddr string `json:"raftAddr"`
	}

	if err := json.NewDecoder(r.Body).Decode(&joinRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := n.Consensus.Join(joinRequest.NodeID, joinRequest.RaftAddr); err != nil {
		http.Error(w, "Failed to join node: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "joined"})
}

// Stop gracefully shuts down the node
func (n *Node) Stop() error {
	log.Printf("Stopping node %s...", n.NodeID)

	if n.Consensus != nil {
		if err := n.Consensus.Shutdown(); err != nil {
			log.Printf("Error shutting down Raft: %v", err)
		}
	}

	if n.Server != nil {
		return n.Server.Close()
	}

	return nil
}

// Helper functions
func getPortFromString(portStr string) int {
	switch portStr {
	case "8000":
		return 8000
	case "8001":
		return 8001
	case "8002":
		return 8002
	default:
		return 8000
	}
}

func getHTTPPortFromRaftAddr(raftAddr string) string {
	// Convert Raft address back to HTTP port
	// Raft ports are HTTP ports + 1000
	parts := strings.Split(raftAddr, ":")
	if len(parts) != 2 {
		return "8000"
	}

	switch parts[1] {
	case "9000":
		return "8000"
	case "9001":
		return "8001"
	case "9002":
		return "8002"
	default:
		return "8000"
	}
}
