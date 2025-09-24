package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"distributedfs/node"
)

func main() {
	port := "8001"
	nodeID := "node-" + port
	isBootstrap := false // replicas never bootstrap

	// Create node
	n, err := node.NewNode(port, nodeID, isBootstrap)
	if err != nil {
		log.Fatalf("❌ Failed to create replica node: %v", err)
	}

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("🛑 Shutting down replica node...")
		if err := n.Stop(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		os.Exit(0)
	}()

	log.Printf("🚀 Replica node %s running on port %s", nodeID, port)
	if err := n.Start(); err != nil {
		log.Fatalf("❌ Failed to start replica node: %v", err)
	}
}
