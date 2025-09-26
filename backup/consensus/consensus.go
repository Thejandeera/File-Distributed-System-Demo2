package consensus

/*
import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// consensus/consensus.go - Replace your existing consensus.go with this
package consensus

import (
"bytes"
"encoding/json"
"fmt"
"io"
"log"
"net"
"os"
"path/filepath"
"sync"
"time"

"github.com/hashicorp/raft"
raftboltdb "github.com/hashicorp/raft-boltdb"
)

// RaftConsensus manages the Raft cluster and applies commands to the FSM
type RaftConsensus struct {
	raft      *raft.Raft
	fsm       *FileFSM
	transport *raft.NetworkTransport
	mu        sync.RWMutex
	nodeID    string
}

// Command represents operations that can be applied to the FSM
type Command struct {
	Op       string `json:"op"`
	Filename string `json:"filename"`
	Data     []byte `json:"data,omitempty"`
}

// FileFSM implements the raft.FSM interface
type FileFSM struct {
	storagePath string
	mu          sync.Mutex
}

// Apply applies a Raft log entry to the file system state machine
func (f *FileFSM) Apply(logEntry *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	var cmd Command
	if err := json.Unmarshal(logEntry.Data, &cmd); err != nil {
		log.Printf("Failed to unmarshal command: %v", err)
		return fmt.Errorf("failed to unmarshal command: %v", err)
	}

	switch cmd.Op {
	case "upload":
		return f.applyUpload(cmd.Filename, cmd.Data)
	case "delete":
		return f.applyDelete(cmd.Filename)
	default:
		return fmt.Errorf("unknown command operation: %s", cmd.Op)
	}
}

// applyUpload saves a file to the file system
func (f *FileFSM) applyUpload(filename string, data []byte) error {
	filePath := filepath.Join(f.storagePath, filename)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", filename, err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write file %s: %v", filename, err)
	}

	log.Printf("FSM: File %s uploaded successfully", filename)
	return nil
}

// applyDelete removes a file from the file system
func (f *FileFSM) applyDelete(filename string) error {
	filePath := filepath.Join(f.storagePath, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", filename)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file %s: %v", filename, err)
	}

	log.Printf("FSM: File %s deleted successfully", filename)
	return nil
}

// Snapshot returns a point-in-time snapshot of the FSM state
func (f *FileFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// For simplicity, we'll create a basic snapshot
	// In production, you'd want to create a proper archive of all files
	files, err := os.ReadDir(f.storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read storage directory: %v", err)
	}

	snapshot := &FileFSMSnapshot{
		storagePath: f.storagePath,
		files:       make(map[string][]byte),
	}

	// Read all files into memory for the snapshot
	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(f.storagePath, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				log.Printf("Warning: failed to read file %s for snapshot: %v", file.Name(), err)
				continue
			}
			snapshot.files[file.Name()] = data
		}
	}

	return snapshot, nil
}

// Restore restores the FSM state from a snapshot
func (f *FileFSM) Restore(snapshot io.ReadCloser) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	defer snapshot.Close()

	// Clear existing files
	files, err := os.ReadDir(f.storagePath)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() {
				os.Remove(filepath.Join(f.storagePath, file.Name()))
			}
		}
	}

	// Decode and restore files from snapshot
	decoder := json.NewDecoder(snapshot)
	var snapshotData map[string][]byte
	if err := decoder.Decode(&snapshotData); err != nil {
		return fmt.Errorf("failed to decode snapshot: %v", err)
	}

	for filename, data := range snapshotData {
		filePath := filepath.Join(f.storagePath, filename)
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			log.Printf("Warning: failed to restore file %s: %v", filename, err)
		}
	}

	log.Println("FSM: State restored from snapshot")
	return nil
}

// FileFSMSnapshot represents a point-in-time snapshot
type FileFSMSnapshot struct {
	storagePath string
	files       map[string][]byte
}

// Persist writes the snapshot to the sink
func (s *FileFSMSnapshot) Persist(sink raft.SnapshotSink) error {
	defer sink.Close()

	encoder := json.NewEncoder(sink)
	return encoder.Encode(s.files)
}

// Release is called when the snapshot is no longer needed
func (s *FileFSMSnapshot) Release() {
	// Nothing to release in our simple implementation
}

// NewRaftConsensus creates a new Raft consensus instance
func NewRaftConsensus(nodeID, raftAddr, raftDir, storagePath string) (*RaftConsensus, error) {
	// Create raft directory
	if err := os.MkdirAll(raftDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create raft directory: %v", err)
	}

	// Create storage directory for FSM
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %v", err)
	}

	// Initialize FSM
	fsm := &FileFSM{
		storagePath: storagePath,
	}

	// Setup Raft configuration
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)
	config.LogLevel = "INFO"

	// Setup Raft log store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-log.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to create log store: %v", err)
	}

	// Setup Raft stable store
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft-stable.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to create stable store: %v", err)
	}

	// Setup snapshot store
	snapshotStore, err := raft.NewFileSnapshotStore(raftDir, 3, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot store: %v", err)
	}

	// Setup Raft transport
	addr, err := net.ResolveTCPAddr("tcp", raftAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %v", err)
	}

	transport, err := raft.NewTCPTransport(raftAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %v", err)
	}

	// Create Raft instance
	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, fmt.Errorf("failed to create raft instance: %v", err)
	}

	rc := &RaftConsensus{
		raft:      r,
		fsm:       fsm,
		transport: transport,
		nodeID:    nodeID,
	}

	return rc, nil
}

// Bootstrap initializes a new Raft cluster
func (rc *RaftConsensus) Bootstrap() error {
	configuration := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      raft.ServerID(rc.nodeID),
				Address: rc.transport.LocalAddr(),
			},
		},
	}
	return rc.raft.BootstrapCluster(configuration).Error()
}

// Join adds this node to an existing cluster
func (rc *RaftConsensus) Join(nodeID, addr string) error {
	log.Printf("Attempting to join cluster as node %s at %s", nodeID, addr)

	configFuture := rc.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return fmt.Errorf("failed to get raft configuration: %v", err)
	}

	for _, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				log.Printf("Node %s already member of cluster, ignoring join request", nodeID)
				return nil
			}

			future := rc.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("failed to remove existing server: %v", err)
			}
		}
	}

	addFuture := rc.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if err := addFuture.Error(); err != nil {
		return fmt.Errorf("failed to add voter: %v", err)
	}

	log.Printf("Node %s joined successfully", nodeID)
	return nil
}

// ApplyCommand applies a command to the Raft log
func (rc *RaftConsensus) ApplyCommand(op, filename string, data []byte) error {
	if rc.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	cmd := Command{
		Op:       op,
		Filename: filename,
		Data:     data,
	}

	cmdBytes, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %v", err)
	}

	future := rc.raft.Apply(cmdBytes, 10*time.Second)
	return future.Error()
}

// IsLeader returns true if this node is the current leader
func (rc *RaftConsensus) IsLeader() bool {
	return rc.raft.State() == raft.Leader
}

// GetLeader returns the current leader's address
func (rc *RaftConsensus) GetLeader() string {
	_, leader := rc.raft.LeaderWithID()
	return string(leader)
}

// GetState returns the current Raft state
func (rc *RaftConsensus) GetState() string {
	return rc.raft.State().String()
}

// GetStats returns Raft statistics
func (rc *RaftConsensus) GetStats() map[string]string {
	return rc.raft.Stats()
}

// Shutdown gracefully shuts down the Raft node
func (rc *RaftConsensus) Shutdown() error {
	return rc.raft.Shutdown().Error()
}
*/
