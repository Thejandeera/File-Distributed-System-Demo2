package main

import (
	"distributedfs/node"
	"log"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	node := node.NewNode(port)

	log.Printf("🚀 Starting distributed file system node on port %s", port)

	if err := node.Start(); err != nil {
		log.Fatalf("❌ Failed to start node: %v", err)
	}
}
