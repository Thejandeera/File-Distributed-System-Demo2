package main

import (
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
	"strings"
)

const storagePath = "./storage_data"
const quotaLimit = 100 * 1024 * 1024 // 100 MB

var selfPort string

func main() {
	selfPort = os.Getenv("PORT")
	if selfPort == "" {
		selfPort = "8000"
	}

	if _, err := os.Stat(storagePath); os.IsNotExist(err) {
		os.Mkdir(storagePath, os.ModePerm)
	}

	// Start background services
	go time_sync.SimulateLogicalClocks()
	go time_sync.SyncClock()
	go consensus.StartRaftElection(selfPort)
	go recoverMissingFiles()
	fault.StartHeartbeat(selfPort)

	// Define API routes
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/download", downloadHandler)
	http.HandleFunc("/files", filesHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/stats", statsHandler)
	http.HandleFunc("/leader", leaderHandler)
	http.HandleFunc("/fileinfo", fileInfoHandler)

	log.Printf("🟢 Node running on port %s\n", selfPort)
	log.Fatal(http.ListenAndServe(":"+selfPort, nil))
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if !consensus.IsLeader(selfPort) {
		http.Error(w, "❌ I'm not the leader", http.StatusForbidden)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "❌ Failed to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	dstPath := filepath.Join(storagePath, header.Filename)

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

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	filename := r.URL.Query().Get("name")
	if filename == "" {
		http.Error(w, "Missing filename", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, filepath.Join(storagePath, filename))
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	files, _ := os.ReadDir(storagePath)
	var names []string
	for _, f := range files {
		if !f.IsDir() {
			names = append(names, f.Name())
		}
	}
	json.NewEncoder(w).Encode(names)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	name := r.URL.Query().Get("name")
	os.Remove(filepath.Join(storagePath, name))
	w.WriteHeader(http.StatusOK)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	files, _ := os.ReadDir(storagePath)
	var totalSize int64
	for _, f := range files {
		if info, err := os.Stat(filepath.Join(storagePath, f.Name())); err == nil {
			totalSize += info.Size()
		}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"totalFiles": len(files),
		"totalBytes": totalSize,
		"quotaBytes": quotaLimit,
	})
}

func leaderHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Write([]byte("Current Leader: " + consensus.GetLeader()))
}

func fileInfoHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	filename := r.URL.Query().Get("name")
	if filename == "" {
		http.Error(w, "Missing filename", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(storagePath, filename)

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

// Replica recovery system
func recoverMissingFiles() {
	peers := []string{"http://localhost:8000", "http://localhost:8001", "http://localhost:8002"}

	for _, peer := range peers {
		if strings.Contains(peer, selfPort) {
			continue
		}

		resp, err := http.Get(peer + "/files")
		if err != nil {
			log.Printf("❌ Cannot fetch files from %s: %v\n", peer, err)
			continue
		}
		defer resp.Body.Close()

		var remoteFiles []string
		if err := json.NewDecoder(resp.Body).Decode(&remoteFiles); err != nil {
			log.Printf("❌ Cannot parse files from %s: %v\n", peer, err)
			continue
		}

		localFiles, _ := os.ReadDir(storagePath)
		localSet := make(map[string]bool)
		for _, f := range localFiles {
			if !f.IsDir() {
				localSet[f.Name()] = true
			}
		}

		for _, file := range remoteFiles {
			if !localSet[file] {
				log.Printf("🔄 Recovering missing file: %s\n", file)
				downloadFile(peer, file)
			}
		}
		break
	}
}

func downloadFile(peerURL, filename string) {
	resp, err := http.Get(peerURL + "/download?name=" + filename)
	if err != nil {
		log.Printf("❌ Failed to download %s: %v\n", filename, err)
		return
	}
	defer resp.Body.Close()

	dstPath := filepath.Join(storagePath, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		log.Printf("❌ Failed to create file %s: %v\n", filename, err)
		return
	}
	defer dst.Close()

	_, err = io.Copy(dst, resp.Body)
	if err != nil {
		log.Printf("❌ Failed to save file %s: %v\n", filename, err)
		return
	}

	log.Printf("✅ Recovered file: %s\n", filename)
}
