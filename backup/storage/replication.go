package storage

import (
	"bytes"
	"distributedfs/config"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"time"
)

// Track replicated files to avoid redundant replication
var replicatedFiles = make(map[string]bool)
var repMu sync.Mutex

// ReplicateToPeers triggers file replication to all configured peers
func ReplicateToPeers(filename, filePath string) {
	repMu.Lock()
	if replicatedFiles[filename] {
		repMu.Unlock()
		return // Already replicated
	}
	replicatedFiles[filename] = true
	repMu.Unlock()

	for _, peer := range config.GetPeers() {
		go func(p string) {
			// Optional small delay to avoid overwhelming network
			time.Sleep(500 * time.Millisecond)
			replicateFileToPeer(p, filename, filePath)
		}(peer)
	}
}

// replicateFileToPeer uploads a file to a peer if needed
func replicateFileToPeer(peer, filename, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("❌ Error opening file for %s: %v\n", peer, err)
		return
	}
	defer file.Close()

	// Step 1: Check if the replica already has a newer version
	shouldReplicate, err := shouldReplicateFile(peer, filename, filePath)
	if err != nil {
		fmt.Printf("❌ Error checking existing file on %s: %v\n", peer, err)
		return
	}
	if !shouldReplicate {
		fmt.Printf("⏩ Skipping replication for '%s' to %s (newer file exists)\n", filename, peer)
		return
	}

	// Step 2: Perform multipart upload
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		fmt.Printf("❌ Error creating form part for %s: %v\n", peer, err)
		return
	}

	if _, err := io.Copy(part, file); err != nil {
		fmt.Printf("❌ Error copying file content to part for %s: %v\n", peer, err)
		return
	}

	writer.Close()

	req, err := http.NewRequest("POST", peer+"/upload", &buf)
	if err != nil {
		fmt.Printf("❌ Error creating request for %s: %v\n", peer, err)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("❌ Replication failed to %s: %v\n", peer, err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("📤 Replicated '%s' to %s → [%d %s]\n", filename, peer, resp.StatusCode, resp.Status)
}

// shouldReplicateFile checks whether the file should be replicated based on timestamps
func shouldReplicateFile(peer, filename, filePath string) (bool, error) {
	// Request file info from the peer
	url := fmt.Sprintf("%s/fileinfo?name=%s", peer, filename)
	resp, err := http.Get(url)
	if err != nil {
		// If peer not reachable or file info not available, assume we should replicate
		return true, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// File does not exist on peer, replicate
		return true, nil
	}

	// Parse file info
	var data struct {
		ModTime int64 `json:"modTime"`
		Size    int64 `json:"size"`
	}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		// If error parsing, assume replicate
		return true, nil
	}

	// Compare timestamps
	localInfo, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	localModTime := localInfo.ModTime().Unix()
	peerModTime := data.ModTime

	// Only replicate if local file is newer
	return localModTime > peerModTime, nil
}
