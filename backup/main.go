package main

import (
	"distributedfs/node"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	port := "8002"
	nodeID := "node-" + port
	isBootstrap := false // replica nodes should never bootstrap

	// Create the node
	n, err := node.NewNode(port, nodeID, isBootstrap)
	if err != nil {
		log.Fatalf("❌ Failed to create replica node: %v", err)
	}

	// Handle graceful shutdown
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
		log.Fatalf("❌ Failed to start replica node: %v", err)
	}
}
