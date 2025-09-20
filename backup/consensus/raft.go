package consensus

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var (
	leader        string
	leaderMutex   sync.Mutex
	nodes         = []string{"8000", "8001", "8002"}
	lastHeartbeat time.Time
)

// StartRaftElection starts the leader election and monitoring process
func StartRaftElection(selfPort string) {
	go monitorHeartbeat()
	go func() {
		rand.Seed(time.Now().UnixNano())
		for {
			time.Sleep(10 * time.Second)

			leaderMutex.Lock()
			if leader == "" {
				// No leader, trigger election
				elected := nodes[rand.Intn(len(nodes))]
				leader = elected
				fmt.Printf("👑 [Raft] Node %s elected as leader\n", elected)
			}
			leaderMutex.Unlock()
		}
	}()

	// Also, start sending heartbeat if this node becomes leader
	go func() {
		for {
			time.Sleep(3 * time.Second)

			if IsLeader(selfPort) {
				sendHeartbeat()
			}
		}
	}()
}

// IsLeader checks if current node is the leader
func IsLeader(selfPort string) bool {
	leaderMutex.Lock()
	defer leaderMutex.Unlock()
	return leader == selfPort
}

// GetLeader returns the current leader
func GetLeader() string {
	leaderMutex.Lock()
	defer leaderMutex.Unlock()
	return leader
}

// simulate heartbeat being sent
func sendHeartbeat() {
	leaderMutex.Lock()
	lastHeartbeat = time.Now()
	leaderMutex.Unlock()
}

// monitorHeartbeat watches for leader failure
func monitorHeartbeat() {
	for {
		time.Sleep(5 * time.Second)

		leaderMutex.Lock()
		elapsed := time.Since(lastHeartbeat)
		leaderMutex.Unlock()

		if elapsed > 8*time.Second || GetLeader() == "" {
			fmt.Println("⚡ [Raft] Leader missing! Starting election...")
			startElection()
		}
	}
}

// startElection randomly elects a new leader
func startElection() {
	leaderMutex.Lock()
	defer leaderMutex.Unlock()

	elected := nodes[rand.Intn(len(nodes))]
	leader = elected
	fmt.Printf("👑 [Raft] Node %s elected as leader (new election)\n", elected)
}

// HeartbeatHandler for receiving heartbeat pings
func HeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	sendHeartbeat()
	w.WriteHeader(http.StatusOK)
}
