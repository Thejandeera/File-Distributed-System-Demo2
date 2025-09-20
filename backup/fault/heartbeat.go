package fault

import (
	"log"
	"net/http"
	"time"
)

// List of peers to check
var peers = []string{
	"http://localhost:8000",
	"http://localhost:8001",
	"http://localhost:8002",
}

// StartHeartbeat periodically pings other servers to check if they're alive
func StartHeartbeat(selfPort string) {
	go func() {
		for {
			for _, peer := range peers {
				if peerContainsSelf(peer, selfPort) {
					continue // Skip checking self
				}

				checkPeerHealth(peer)
			}
			time.Sleep(5 * time.Second) // ⏳ Every 5 seconds
		}
	}()
}

// Check if peer contains our own port
func peerContainsSelf(peer, selfPort string) bool {
	return peer == "http://localhost:"+selfPort
}

// Check if a peer is healthy
func checkPeerHealth(peer string) {
	resp, err := http.Get(peer + "/health")
	if err != nil {
		log.Printf("❌ Heartbeat failed: %s unreachable\n", peer)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("✅ Heartbeat OK: %s\n", peer)
	} else {
		log.Printf("⚠️ Heartbeat warning: %s responded with status %d\n", peer, resp.StatusCode)
	}
}
