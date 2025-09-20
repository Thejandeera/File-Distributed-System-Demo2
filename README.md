# 📂 Distributed File System

A fault-tolerant, time-synchronized, and replicated distributed file system developed as part of our Distributed Systems coursework.


---

## 🛠️ Tech Stack

- **Backend Language**: Go (Golang)
- **Frontend**: Next.js 15 (React)
- **Communication**: REST APIs over HTTP
- **Replication**: Leader-based file replication
- **Consensus**: Raft-based Leader Election
- **Time Sync**: NTP Synchronization + Lamport Logical Clocks
- **Fault Tolerance**: Auto-replication and file recovery

---

## 📁 Folder Structure

```
distributedfs/
├── config/          # Configuration parameters
├── consensus/       # Raft consensus and election logic
├── fault/           # Failure detection and recovery modules
├── replica1/        # Replica node 1 (runs main.go)
├── replica2/        # Replica node 2 (runs main.go)
├── storage/         # File management & replication
├── storage_data/    # Directory for uploaded files
├── time_sync/       # NTP sync + Lamport Clock
├── go.mod / go.sum  # Go module files
├── main.go          # Main backend application
├── README.md        # Backend README

frontend/
├── app/             # App routes (Next.js)
├── public/          # Static files
├── node_modules/    # Dependencies
├── .next/           # Build output
├── README.md        # Frontend README
├── package.json     # Project metadata & scripts
├── tsconfig.json    # TypeScript config
```

---

## ⚙️ Setup & Run

### 1. Clone the Repository

```bash
git clone https://github.com/<your-username>/distributed-file-system.git
cd distributed-file-system
```

---

### 2. Run Backend Replicas

Each node runs the same `main.go` with different ports.

#### Terminal 1 (Main Leader)
```bash
cd distributedfs
$env:PORT="8000"
go run main.go
```

#### Terminal 2 (Replica 1)
```bash
cd distributedfs
$env:PORT="8001"
go run main.go
```

#### Terminal 3 (Replica 2)
```bash
cd distributedfs
$env:PORT="8002"
go run main.go
```

---

### 3. Run the Frontend

```bash
cd frontend
npm install
npm run dev
```

App will be hosted at: **http://localhost:3000**

---

## 🧪 Features

- ✅ File Upload, Download, Delete
- ✅ Replication from Leader to Replicas
- ✅ Fault detection & Heartbeat monitoring
- ✅ Leader election via Raft simulation
- ✅ Time synchronization with NTP & Lamport clocks
- ✅ Auto-recovery of missing files in replicas

---

## 🧪 Testing Suggestions

- Stop a node and observe heartbeat + recovery.
- Upload a file via UI; verify all 3 nodes receive it.
- Delete file via UI and validate deletion across replicas.
- Test leader crash and auto-election handling.

---

## 🤝 Collaboration Notes

- Branch naming: `feature/<module>` (e.g., `feature/replication`)
- Push with descriptive commit messages.
- Code formatting: Prettier + ESLint recommended for frontend.

---

## ✨ Future Enhancements

- Hide delete/download buttons for replicas when offline.
- Support for file versioning (timestamps, conflict resolution UI).
- Peer discovery & auto join for new nodes.
- Stronger consistency model (quorum, voting).
- Support uploads to replicas when leader is down (P2P fallback).

---

## 📩 Contact & Support

Please use GitHub Issues for bug reports or queries.


