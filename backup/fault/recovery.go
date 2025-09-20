package fault

import (
	"distributedfs/config"
	"distributedfs/consensus"
	"distributedfs/storage"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RecoveryManager handles node recovery and data consistency
type RecoveryManager struct {
	selfPort     string
	storagePath  string
	recoveryMu   sync.Mutex
	isRecovering bool
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(selfPort, storagePath string) *RecoveryManager {
	return &RecoveryManager{
		selfPort:     selfPort,
		storagePath:  storagePath,
		isRecovering: false,
	}
}

// StartRecoveryProcess starts the background recovery process
func (rm *RecoveryManager) StartRecoveryProcess() {
	go rm.periodicRecovery()
	go rm.monitorNodeHealth()
}

// periodicRecovery runs recovery checks periodically
func (rm *RecoveryManager) periodicRecovery() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !rm.isRecovering {
			go rm.performRecovery()
		}
	}
}

// monitorNodeHealth monitors the health of other nodes
func (rm *RecoveryManager) monitorNodeHealth() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		rm.checkNodeHealth()
	}
}

// performRecovery performs comprehensive recovery operations
func (rm *RecoveryManager) performRecovery() {
	rm.recoveryMu.Lock()
	if rm.isRecovering {
		rm.recoveryMu.Unlock()
		return
	}
	rm.isRecovering = true
	rm.recoveryMu.Unlock()

	defer func() {
		rm.recoveryMu.Lock()
		rm.isRecovering = false
		rm.recoveryMu.Unlock()
	}()

	log.Println("🔄 Starting recovery process...")

	// 1. Recover missing files
	rm.recoverMissingFiles()

	// 2. Verify file integrity
	rm.verifyFileIntegrity()

	// 3. Sync with other nodes
	rm.syncWithPeers()

	log.Println("✅ Recovery process completed")
}

// recoverMissingFiles recovers files that are missing from this node
func (rm *RecoveryManager) recoverMissingFiles() {
	peers := rm.getAvailablePeers()
	if len(peers) == 0 {
		log.Println("⚠️ No peers available for recovery")
		return
	}

	for _, peer := range peers {
		rm.recoverFromPeer(peer)
	}
}

// getAvailablePeers returns a list of available peers
func (rm *RecoveryManager) getAvailablePeers() []string {
	var availablePeers []string

	for _, peer := range config.GetPeers() {
		if rm.isPeerAvailable(peer) {
			availablePeers = append(availablePeers, peer)
		}
	}

	return availablePeers
}

// isPeerAvailable checks if a peer is available
func (rm *RecoveryManager) isPeerAvailable(peer string) bool {
	resp, err := http.Get(peer + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// recoverFromPeer recovers files from a specific peer
func (rm *RecoveryManager) recoverFromPeer(peer string) {
	log.Printf("🔄 Recovering from peer: %s", peer)

	// Get list of files from peer
	resp, err := http.Get(peer + "/files")
	if err != nil {
		log.Printf("❌ Cannot fetch files from %s: %v", peer, err)
		return
	}
	defer resp.Body.Close()

	var remoteFiles []string
	if err := json.NewDecoder(resp.Body).Decode(&remoteFiles); err != nil {
		log.Printf("❌ Cannot parse files from %s: %v", peer, err)
		return
	}

	// Get local files
	localFiles, _ := os.ReadDir(rm.storagePath)
	localSet := make(map[string]bool)
	for _, f := range localFiles {
		if !f.IsDir() {
			localSet[f.Name()] = true
		}
	}

	// Download missing files
	for _, file := range remoteFiles {
		if !localSet[file] {
			log.Printf("🔄 Recovering missing file: %s", file)
			rm.downloadFile(peer, file)
		}
	}
}

// downloadFile downloads a file from a peer
func (rm *RecoveryManager) downloadFile(peerURL, filename string) error {
	resp, err := http.Get(peerURL + "/download?name=" + filename)
	if err != nil {
		log.Printf("❌ Failed to download %s: %v", filename, err)
		return err
	}
	defer resp.Body.Close()

	dstPath := filepath.Join(rm.storagePath, filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		log.Printf("❌ Failed to create file %s: %v", filename, err)
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, resp.Body)
	if err != nil {
		log.Printf("❌ Failed to save file %s: %v", filename, err)
		return err
	}

	log.Printf("✅ Recovered file: %s", filename)
	return nil
}

// verifyFileIntegrity verifies the integrity of local files
func (rm *RecoveryManager) verifyFileIntegrity() {
	files, err := os.ReadDir(rm.storagePath)
	if err != nil {
		log.Printf("❌ Cannot read storage directory: %v", err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(rm.storagePath, file.Name())
		if !rm.isFileValid(filePath) {
			log.Printf("⚠️ Corrupted file detected: %s", file.Name())
			rm.repairFile(filePath)
		}
	}
}

// isFileValid checks if a file is valid
func (rm *RecoveryManager) isFileValid(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()

	// Try to read a small portion to check if file is accessible
	buffer := make([]byte, 1)
	_, err = file.Read(buffer)
	return err == nil
}

// repairFile attempts to repair a corrupted file
func (rm *RecoveryManager) repairFile(filePath string) {
	filename := filepath.Base(filePath)

	// Try to get a good copy from peers
	peers := rm.getAvailablePeers()
	for _, peer := range peers {
		if rm.downloadFile(peer, filename) == nil {
			log.Printf("✅ Repaired file: %s", filename)
			return
		}
	}

	log.Printf("❌ Could not repair file: %s", filename)
}

// syncWithPeers synchronizes with other nodes
func (rm *RecoveryManager) syncWithPeers() {
	if !consensus.IsLeader(rm.selfPort) {
		return
	}

	log.Println("🔄 Syncing with peers...")

	// Trigger replication for all local files
	files, err := os.ReadDir(rm.storagePath)
	if err != nil {
		log.Printf("❌ Cannot read storage directory: %v", err)
		return
	}

	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(rm.storagePath, file.Name())
			go storage.ReplicateToPeers(file.Name(), filePath)
		}
	}
}

// checkNodeHealth checks the health of other nodes
func (rm *RecoveryManager) checkNodeHealth() {
	for _, peer := range config.GetPeers() {
		if !rm.isPeerAvailable(peer) {
			log.Printf("⚠️ Node %s is not responding", peer)
		}
	}
}

// HandleNodeFailure handles the failure of a node
func (rm *RecoveryManager) HandleNodeFailure(failedNode string) {
	log.Printf("🚨 Node failure detected: %s", failedNode)

	// If we're the leader, trigger immediate recovery
	if consensus.IsLeader(rm.selfPort) {
		go rm.performRecovery()
	}
}

// GetRecoveryStatus returns the current recovery status
func (rm *RecoveryManager) GetRecoveryStatus() map[string]interface{} {
	rm.recoveryMu.Lock()
	defer rm.recoveryMu.Unlock()

	return map[string]interface{}{
		"isRecovering": rm.isRecovering,
		"selfPort":     rm.selfPort,
		"storagePath":  rm.storagePath,
	}
}
