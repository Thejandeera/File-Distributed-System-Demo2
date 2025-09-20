package node

import (
	"distributedfs/config"
	"distributedfs/consensus"
	"distributedfs/fault"
	"distributedfs/storage"
	"distributedfs/time_sync"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)


// Node represents a distributed file system node
type Node struct {
	Port           string
	StoragePath    string
	QuotaLimit     int64
	RecoveryMgr    *fault.RecoveryManager
	FileMgr        *storage.FileManager
	LeaderMgr      *consensus.LeaderManager
	Server         *http.Server
}

// NewNode creates a new node instance
func NewNode(port string) *Node {
	// Initialize configuration
	config.InitializeConfig()
	
	storagePath := config.GetStoragePath()
	quotaLimit := config.GetQuotaLimit()
	
	// Create managers
	recoveryMgr := fault.NewRecoveryManager(port, storagePath)
	fileMgr := storage.NewFileManager(storagePath, quotaLimit)
	leaderMgr := consensus.NewLeaderManager([]string{"8000", "8001", "8002"})
	
	return &Node{
		Port:        port,
		StoragePath: storagePath,
		QuotaLimit:  quotaLimit,
		RecoveryMgr: recoveryMgr,
		FileMgr:     fileMgr,
		LeaderMgr:   leaderMgr,
	}
}

// Start initializes and starts the node
func (n *Node) Start() error {
	// Ensure storage directory exists
	if err := os.MkdirAll(n.StoragePath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create storage directory: %v", err)
	}

	// Start background services
	go time_sync.SimulateLogicalClocks()
	go time_sync.SyncClock()
	go consensus.StartRaftElection(n.Port)
	go n.RecoveryMgr.StartRecoveryProcess()
	go fault.StartHeartbeat(n.Port)
	go n.startCleanupRoutine()

	// Setup HTTP routes
	mux := http.NewServeMux()
	n.setupRoutes(mux)

	// Create HTTP server
	n.Server = &http.Server{
		Addr:    ":" + n.Port,
		Handler: mux,
	}

	log.Printf("🟢 Node running on port %s", n.Port)
	return n.Server.ListenAndServe()
}

// setupRoutes configures HTTP routes
func (n *Node) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/upload", n.uploadHandler)
	mux.HandleFunc("/download", n.downloadHandler)
	mux.HandleFunc("/files", n.filesHandler)
	mux.HandleFunc("/delete", n.deleteHandler)
	mux.HandleFunc("/health", n.healthCheck)
	mux.HandleFunc("/stats", n.statsHandler)
	mux.HandleFunc("/leader", n.leaderHandler)
	mux.HandleFunc("/fileinfo", n.fileInfoHandler)
	mux.HandleFunc("/recovery/status", n.recoveryStatusHandler)
	mux.HandleFunc("/config", n.configHandler)
}

// enableCORS sets CORS headers
func (n *Node) enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE, PUT")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// uploadHandler handles file uploads
func (n *Node) uploadHandler(w http.ResponseWriter, r *http.Request) {
	n.enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if !consensus.IsLeader(n.Port) {
		http.Error(w, "❌ I'm not the leader", http.StatusForbidden)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "❌ Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dstPath := filepath.Join(n.StoragePath, header.Filename)

	// Conflict detection
	if _, err := os.Stat(dstPath); err == nil {
		existingInfo, _ := os.Stat(dstPath)
		now := time_sync.GetCorrectedTime()
		if now.Before(existingInfo.ModTime()) {
			log.Println("⚡ Conflict detected: Incoming file older, rejecting upload")
			http.Error(w, "❌ Conflict: Existing file is newer", http.StatusConflict)
			return
		}
		log.Println("⚡ Conflict detected: Overwriting with newer upload")
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "❌ Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "❌ Failed to write file", http.StatusInternalServerError)
		return
	}

	go storage.ReplicateToPeers(header.Filename, dstPath)

	fmt.Fprintf(w, "✅ File uploaded: %s", header.Filename)
}

// downloadHandler handles file downloads
func (n *Node) downloadHandler(w http.ResponseWriter, r *http.Request) {
	n.enableCORS(w)
	filename := r.URL.Query().Get("name")
	if filename == "" {
		http.Error(w, "Missing filename", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, filepath.Join(n.StoragePath, filename))
}

// filesHandler returns list of files
func (n *Node) filesHandler(w http.ResponseWriter, r *http.Request) {
	n.enableCORS(w)
	files, _ := os.ReadDir(n.StoragePath)
	var names []string
	for _, f := range files {
		if !f.IsDir() {
			names = append(names, f.Name())
		}
	}
	json.NewEncoder(w).Encode(names)
}

// deleteHandler handles file deletion
func (n *Node) deleteHandler(w http.ResponseWriter, r *http.Request) {
	n.enableCORS(w)
	name := r.URL.Query().Get("name")
	os.Remove(filepath.Join(n.StoragePath, name))
	w.WriteHeader(http.StatusOK)
}

// healthCheck returns node health status
func (n *Node) healthCheck(w http.ResponseWriter, r *http.Request) {
	n.enableCORS(w)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// statsHandler returns storage statistics
func (n *Node) statsHandler(w http.ResponseWriter, r *http.Request) {
	n.enableCORS(w)
	files, _ := os.ReadDir(n.StoragePath)
	var totalSize int64
	for _, f := range files {
		if info, err := os.Stat(filepath.Join(n.StoragePath, f.Name())); err == nil {
			totalSize += info.Size()
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"totalFiles": len(files),
		"totalBytes": totalSize,
		"quotaBytes": n.QuotaLimit,
	})
}

// leaderHandler returns current leader information
func (n *Node) leaderHandler(w http.ResponseWriter, r *http.Request) {
	n.enableCORS(w)
	w.Write([]byte("Current Leader: " + consensus.GetLeader()))
}

// fileInfoHandler returns file information
func (n *Node) fileInfoHandler(w http.ResponseWriter, r *http.Request) {
	n.enableCORS(w)

	filename := r.URL.Query().Get("name")
	if filename == "" {
		http.Error(w, "Missing filename", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(n.StoragePath, filename)

	info, err := os.Stat(fullPath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"modTime": info.ModTime().Unix(),
		"size":    info.Size(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// recoveryStatusHandler returns recovery status
func (n *Node) recoveryStatusHandler(w http.ResponseWriter, r *http.Request) {
	n.enableCORS(w)
	status := n.RecoveryMgr.GetRecoveryStatus()
	json.NewEncoder(w).Encode(status)
}

// configHandler handles configuration requests
func (n *Node) configHandler(w http.ResponseWriter, r *http.Request) {
	n.enableCORS(w)
	
	switch r.Method {
	case "GET":
		config := config.GetConfig()
		json.NewEncoder(w).Encode(config)
	case "PUT":
		var updates map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		if err := config.UpdateConfig(updates); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// startCleanupRoutine starts the cleanup routine
func (n *Node) startCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		n.FileMgr.Cleanup()
	}
}

// Stop gracefully stops the node
func (n *Node) Stop() error {
	if n.Server != nil {
		return n.Server.Close()
	}
	return nil
}

// GetNodeInfo returns node information
func (n *Node) GetNodeInfo() map[string]interface{} {
	return map[string]interface{}{
		"port":        n.Port,
		"storagePath": n.StoragePath,
		"quotaLimit":  n.QuotaLimit,
		"isLeader":    consensus.IsLeader(n.Port),
		"leader":      consensus.GetLeader(),
	}
}
