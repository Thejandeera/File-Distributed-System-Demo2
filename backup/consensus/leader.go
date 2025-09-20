package consensus

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// LeaderInfo contains information about the current leader
type LeaderInfo struct {
	Port        string    `json:"port"`
	ElectedAt   time.Time `json:"electedAt"`
	LastSeen    time.Time `json:"lastSeen"`
	IsHealthy   bool      `json:"isHealthy"`
	VoteCount   int       `json:"voteCount"`
}

// LeaderManager handles leader election and management
type LeaderManager struct {
	currentLeader *LeaderInfo
	leaderMu      sync.RWMutex
	nodes         []string
	electionMu    sync.Mutex
	isElecting    bool
}

// NewLeaderManager creates a new leader manager
func NewLeaderManager(nodes []string) *LeaderManager {
	return &LeaderManager{
		nodes:         nodes,
		currentLeader: nil,
		isElecting:    false,
	}
}

// GetCurrentLeader returns the current leader information
func (lm *LeaderManager) GetCurrentLeader() *LeaderInfo {
	lm.leaderMu.RLock()
	defer lm.leaderMu.RUnlock()
	return lm.currentLeader
}

// SetLeader sets the current leader
func (lm *LeaderManager) SetLeader(port string) {
	lm.leaderMu.Lock()
	defer lm.leaderMu.Unlock()
	
	lm.currentLeader = &LeaderInfo{
		Port:      port,
		ElectedAt: time.Now(),
		LastSeen:  time.Now(),
		IsHealthy: true,
		VoteCount: 1,
	}
	
	log.Printf("👑 New leader elected: %s", port)
}

// UpdateLeaderHealth updates the leader's health status
func (lm *LeaderManager) UpdateLeaderHealth(isHealthy bool) {
	lm.leaderMu.Lock()
	defer lm.leaderMu.Unlock()
	
	if lm.currentLeader != nil {
		lm.currentLeader.IsHealthy = isHealthy
		lm.currentLeader.LastSeen = time.Now()
	}
}

// IsLeader checks if the given port is the current leader
func (lm *LeaderManager) IsLeader(port string) bool {
	lm.leaderMu.RLock()
	defer lm.leaderMu.RUnlock()
	
	return lm.currentLeader != nil && lm.currentLeader.Port == port
}

// GetLeaderPort returns the current leader's port
func (lm *LeaderManager) GetLeaderPort() string {
	lm.leaderMu.RLock()
	defer lm.leaderMu.RUnlock()
	
	if lm.currentLeader != nil {
		return lm.currentLeader.Port
	}
	return ""
}

// StartElection starts a new leader election
func (lm *LeaderManager) StartElection() {
	lm.electionMu.Lock()
	if lm.isElecting {
		lm.electionMu.Unlock()
		return
	}
	lm.isElecting = true
	lm.electionMu.Unlock()

	defer func() {
		lm.electionMu.Lock()
		lm.isElecting = false
		lm.electionMu.Unlock()
	}()

	log.Println("🗳️ Starting leader election...")

	// Simple random election for now
	// In a real implementation, this would use Raft protocol
	selectedNode := lm.selectRandomNode()
	lm.SetLeader(selectedNode)
}

// selectRandomNode selects a random node for leadership
func (lm *LeaderManager) selectRandomNode() string {
	if len(lm.nodes) == 0 {
		return ""
	}
	
	// Simple round-robin selection
	now := time.Now()
	index := int(now.Unix()) % len(lm.nodes)
	return lm.nodes[index]
}

// CheckLeaderHealth checks if the current leader is healthy
func (lm *LeaderManager) CheckLeaderHealth() bool {
	lm.leaderMu.RLock()
	leader := lm.currentLeader
	lm.leaderMu.RUnlock()

	if leader == nil {
		return false
	}

	// Check if leader is still responding
	healthURL := fmt.Sprintf("http://localhost:%s/health", leader.Port)
	resp, err := http.Get(healthURL)
	if err != nil {
		log.Printf("❌ Leader health check failed: %v", err)
		lm.UpdateLeaderHealth(false)
		return false
	}
	defer resp.Body.Close()

	isHealthy := resp.StatusCode == http.StatusOK
	lm.UpdateLeaderHealth(isHealthy)
	return isHealthy
}

// HandleLeaderFailure handles the failure of the current leader
func (lm *LeaderManager) HandleLeaderFailure() {
	log.Println("🚨 Leader failure detected, starting new election...")
	lm.StartElection()
}

// GetLeaderStatus returns the current leader status
func (lm *LeaderManager) GetLeaderStatus() map[string]interface{} {
	lm.leaderMu.RLock()
	defer lm.leaderMu.RUnlock()

	if lm.currentLeader == nil {
		return map[string]interface{}{
			"hasLeader": false,
		}
	}

	return map[string]interface{}{
		"hasLeader":   true,
		"port":        lm.currentLeader.Port,
		"electedAt":   lm.currentLeader.ElectedAt,
		"lastSeen":    lm.currentLeader.LastSeen,
		"isHealthy":   lm.currentLeader.IsHealthy,
		"voteCount":   lm.currentLeader.VoteCount,
	}
}

// LeaderStatusHandler handles HTTP requests for leader status
func (lm *LeaderManager) LeaderStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	status := lm.GetLeaderStatus()
	json.NewEncoder(w).Encode(status)
}

// VoteHandler handles voting in leader elections
func (lm *LeaderManager) VoteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var vote struct {
		Candidate string `json:"candidate"`
		Voter     string `json:"voter"`
	}

	if err := json.NewDecoder(r.Body).Decode(&vote); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Simple voting logic - in a real implementation, this would be more complex
	lm.leaderMu.Lock()
	if lm.currentLeader != nil && lm.currentLeader.Port == vote.Candidate {
		lm.currentLeader.VoteCount++
	}
	lm.leaderMu.Unlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "vote recorded"})
}

// Global leader manager instance
var globalLeaderManager *LeaderManager

// InitializeLeaderManager initializes the global leader manager
func InitializeLeaderManager(nodes []string) {
	globalLeaderManager = NewLeaderManager(nodes)
}

// GetGlobalLeaderManager returns the global leader manager
func GetGlobalLeaderManager() *LeaderManager {
	return globalLeaderManager
}
