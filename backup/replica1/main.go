package main

import (
	"distributedfs/node"
	"log"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	// Create and start node
	node := node.NewNode(port)
	
	log.Printf("🚀 Starting distributed file system replica on port %s", port)
	
	if err := node.Start(); err != nil {
		log.Fatalf("❌ Failed to start replica: %v", err)
	}
}

