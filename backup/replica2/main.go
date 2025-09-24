package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"distributedfs/node"
)

func main() {
	port := "8002" // Replica port
	nodeID := "node-" + port
	isBootstrap := false // Replica nodes never bootstrap

	leaderAddr := "127.0.0.1:9000" // Bootstrap node Raft port

	// Create the node
	n, err := node.NewNode(port, nodeID, isBootstrap)
	if err != nil {
		log.Fatalf("❌ Failed to create replica node: %v", err)
	}

	// Join the cluster
	if err := n.JoinCluster(nodeID, leaderAddr); err != nil {
		log.Printf("⚠️ Failed to join cluster: %v", err)
	} else {
		log.Printf("✅ Replica node %s successfully requested to join cluster via %s", nodeID, leaderAddr)
	}

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("🛑 Shutting down replica gracefully...")
		if err := n.Stop(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		os.Exit(0)
	}()

	log.Printf("🚀 Starting replica node %s on port %s", nodeID, port)

	// Start the node
	if err := n.Start(); err != nil {
		log.Fatalf("❌ Failed to start replica: %v", err)
	}
}
