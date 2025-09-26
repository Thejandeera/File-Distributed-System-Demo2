# Enhanced Distributed File System

A robust, production-ready distributed file system implemented in Go with advanced fault tolerance, automatic recovery, and comprehensive monitoring capabilities.

## 🚀 Overview

This distributed file system provides enterprise-grade file storage and replication across multiple nodes with automatic failover, leader election, and self-healing capabilities. Built with Go and designed for high availability and data consistency.

## ▶️ Run (Raft + HTTP)

### Build once
```powershell
cd backup
go mod tidy
go build -o dfsapi.exe .
```

### Start 3-node cluster (recommended)
```powershell
./start-cluster.ps1
```

### Start nodes manually (alternative)
Run each in a separate terminal from the `backup` directory:
```powershell
./dfsapi.exe --node 0 --http :8081 --cluster 1,:3030;2,:3031;3,:3032
./dfsapi.exe --node 1 --http :8082 --cluster 1,:3030;2,:3031;3,:3032
./dfsapi.exe --node 2 --http :8083 --cluster 1,:3030;2,:3031;3,:3032
```

### Verify
```bash
curl http://localhost:8081/status
curl http://localhost:8082/status
curl http://localhost:8083/status
```

### Upload and fetch files
```bash
# Upload to leader (replace 8081 with the leader's HTTP port)
curl -X POST --data-binary @README.md http://localhost:8081/upload/README.md

# List files (on any node)
curl http://localhost:8082/files

# Download (served from local ./data folder)
curl http://localhost:8083/README.md -o README.copy.md
```

### Windows note
If PowerShell blocks scripts, run the launcher with:
```powershell
powershell -ExecutionPolicy Bypass -File .\start-cluster.ps1
```

## ✨ Key Features

### Core Architecture
- **Unified Node Package**: Eliminates code duplication with shared node implementation
- **Leader-Follower Model**: Automatic leader election with failover capabilities
- **Multi-Node Replication**: Files automatically replicated across all active nodes
- **Fault Tolerance**: Self-healing system with automatic recovery mechanisms

### Advanced Features
- **File Locking**: Concurrent access control to prevent data corruption
- **Quota Management**: Storage limit enforcement with usage monitoring
- **Dynamic Configuration**: Runtime configuration updates via JSON API
- **Integrity Verification**: Automatic file corruption detection and repair
- **Health Monitoring**: Comprehensive node health checks and status reporting
- **Background Services**: Automatic cleanup and maintenance tasks

### API & Monitoring
- **RESTful API**: Complete HTTP API for file operations and system management
- **Real-time Statistics**: Storage usage, file counts, and performance metrics
- **Recovery Status**: Detailed recovery system monitoring and reporting
- **Configuration Management**: Live configuration updates without restarts

## 🏗️ Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│    Node 1   │    │    Node 2   │    │    Node 3   │
│  (Leader)   │◄──►│ (Follower)  │◄──►│ (Follower)  │
│   :8000     │    │   :8001     │    │   :8002     │
└─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │
       └───────────────────┼───────────────────┘
                           │
                    ┌─────────────┐
                    │  Shared     │
                    │  Storage    │
                    │  Layer      │
                    └─────────────┘
```

## 📋 System Requirements

- **Go**: Version 1.19 or higher
- **Operating System**: Windows, Linux, or macOS
- **Network**: Ports 8000, 8001, 8002 available
- **Storage**: Sufficient disk space for file storage and replication

## 🛠️ Installation & Setup

### Prerequisites
```bash
# Ensure Go is installed
go version

# Clone the repository (if applicable)
cd backup

# Install dependencies
go mod tidy
```

### Quick Start

1. **Start the main node** (Terminal 1):
```powershell
$env:PORT="8000"
go run main.go
```

2. **Start replica 1** (Terminal 2):
```powershell
$env:PORT="8001"
go run replica1/main.go
```

3. **Start replica 2** (Terminal 3):
```powershell
$env:PORT="8002"
go run replica2/main.go
```

4. **Verify system health**:
```bash
curl http://localhost:8000/health
curl http://localhost:8001/health
curl http://localhost:8002/health
```

## 📖 API Reference

### Health & Status Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Node health check |
| `/stats` | GET | Storage statistics and metrics |
| `/recovery/status` | GET | Recovery system status |

### Leader Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/leader` | GET | Current leader information |
| `/leader/status` | GET | Detailed leader status |
| `/leader/vote` | POST | Participate in leader election |

### Configuration Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/config` | GET | Get current configuration |
| `/config` | PUT | Update configuration |

### File Operations

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/upload` | POST | Upload file (leader only) |
| `/download` | GET | Download file |
| `/files` | GET | List all files |
| `/fileinfo` | GET | Get file metadata |

## 🧪 Testing Guide

### Basic System Verification

1. **Health Check All Nodes**:
```bash
# Test all nodes are responsive
curl http://localhost:8000/health
curl http://localhost:8001/health
curl http://localhost:8002/health
```

2. **Verify Leader Election**:
```bash
# Check which node is the leader
curl http://localhost:8000/leader
```

### File Operations Testing

1. **Upload Test File**:
```bash
# Create test file
echo "Test content for distributed file system" > test.txt

# Upload to leader only
curl -X POST -F "file=@test.txt" http://localhost:8000/upload
```

2. **Verify Replication**:
```bash
# Check file appears on all nodes
curl http://localhost:8000/files
curl http://localhost:8001/files
curl http://localhost:8002/files
```

3. **Download and Verify**:
```bash
# Download from any node
curl "http://localhost:8001/download?name=test.txt" -o downloaded.txt

# Verify content
cat downloaded.txt
```

### Advanced Feature Testing

1. **Configuration Management**:
```bash
# Get current config
curl http://localhost:8000/config

# Update configuration
curl -X PUT http://localhost:8000/config \
  -H "Content-Type: application/json" \
  -d '{"logLevel": "DEBUG"}'
```

2. **Storage Statistics**:
```bash
# Get detailed storage stats
curl http://localhost:8000/stats | jq
```

3. **Recovery System**:
```bash
# Check recovery status
curl http://localhost:8000/recovery/status
```

### Fault Tolerance Testing

1. **Node Failure Simulation**:
   - Stop one replica (Ctrl+C)
   - Upload files to remaining nodes
   - Restart stopped node
   - Verify automatic recovery

2. **Leader Failover**:
   - Stop current leader
   - Verify new leader election
   - Test file operations on new leader

## 📊 Monitoring & Metrics

### Health Metrics
- Node status and uptime
- Network connectivity between nodes
- Storage capacity and usage
- File integrity status

### Performance Metrics
- Upload/download throughput
- Replication latency
- Recovery time
- API response times

### Example Statistics Response
```json
{
  "totalFiles": 42,
  "totalSize": 1048576,
  "quotaUsed": 0.25,
  "quotaLimit": 4194304,
  "storageUsage": "25.0%",
  "nodeStatus": "healthy",
  "lastBackup": "2025-09-20T10:30:00Z"
}
```

## 🔧 Configuration

### Environment Variables
- `PORT`: Node port (8000, 8001, 8002)
- `STORAGE_PATH`: Custom storage directory
- `LOG_LEVEL`: Logging level (INFO, DEBUG, ERROR)

### Configuration File Format
```json
{
  "nodeId": "node-8000",
  "port": 8000,
  "peers": ["localhost:8001", "localhost:8002"],
  "storageQuota": 4194304,
  "logLevel": "INFO",
  "heartbeatInterval": 5000,
  "recoveryEnabled": true
}
```

## 🛡️ Security Features

- **File Integrity**: SHA-256 checksums for corruption detection
- **Access Control**: Leader-only write operations
- **Input Validation**: Comprehensive parameter validation
- **Error Handling**: Secure error messages without sensitive data

## 🔍 Troubleshooting

### Common Issues

**Port Already in Use**:
```bash
# Check port usage
netstat -an | findstr :8000
# Kill process if needed
taskkill /F /PID <process_id>
```

**Node Connection Issues**:
- Verify firewall settings
- Check network connectivity between nodes
- Ensure all nodes are running

**File Replication Delays**:
- Check network latency between nodes
- Verify storage space availability
- Monitor system logs for errors

**Leader Election Problems**:
- Ensure at least 2 nodes are running
- Check heartbeat connectivity
- Verify configuration consistency

### Log Analysis
```bash
# Check node logs for errors
# Logs are written to console by default
# Look for keywords: ERROR, FATAL, leader, replication
```

## 📈 Performance Tuning

### Optimization Tips
- Adjust heartbeat intervals for your network
- Configure appropriate storage quotas
- Monitor and tune replication delays
- Use SSD storage for better I/O performance

### Scaling Considerations
- Add more replicas for higher availability
- Implement load balancing for read operations
- Consider sharding for large datasets
- Monitor resource usage and scale accordingly

## 🤝 Contributing

### Development Setup
1. Fork the repository
2. Create feature branch
3. Run tests: `go test ./...`
4. Submit pull request

### Code Style
- Follow Go conventions
- Use meaningful variable names
- Add comprehensive error handling
- Include unit tests for new features

## 📜 License

This project is licensed under the MIT License. See LICENSE file for details.

## 📞 Support

For issues and questions:
- Check troubleshooting section
- Review system logs
- Open GitHub issues for bugs
- Contribute improvements via pull requests

---

**Built with ❤️ using Go** | **Production Ready** | **Enterprise Grade**