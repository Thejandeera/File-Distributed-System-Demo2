# 📦 Distributed File System - Backend

This is the **backend implementation** of a basic **Distributed File System (DFS)** built in Go. It supports **file uploads, downloads, listing, deletion**, **replication across multiple replicas**, and **replica health monitoring** via heartbeats.

---

## 🚀 Features

- ✅ File Upload (`/upload`)
- ✅ File Download (`/download?name=filename`)
- ✅ List Files (`/files`)
- ✅ Delete Files (`/delete?name=filename`)
- ✅ File Replication to replicas
- ✅ Heartbeat-based Replica Monitoring (`/health`)
- ✅ CORS enabled for frontend integration

---

## 🗂️ Folder Structure


distributed-file-system/
├── distributedfs/         # Backend servers (Go)
│    ├── main.go
│    ├── consensus/         # Raft leader election
│    ├── storage/           # File replication
│    ├── time_sync/         # NTP + Lamport clocks
│    ├── fault/             # Heartbeats and recovery
│    └── config/
│
├── frontend/               # Next.js frontend
│    └── distributed-ui/
│        ├── app/
│        ├── public/
│        ├── package.json
│        └── README.md
│
├── README.md                # (You're reading it!)
└── go.mod



git clone <your repository link>
cd distributed-file-system


🚀 How to Run
1. Clone the Repository
git clone <your repository link>
cd distributed-file-system

2. Install Backend Dependencies
Inside the distributedfs folder:
cd distributedfs
go mod tidy

3. Start Backend Servers
You must start 3 instances of the backend:

➡️ Open 3 terminals (or Powershells) and run:

Main Node (Port 8000):

$env:PORT="8000"
go run main.go
Replica 1 (Port 8001):

$env:PORT="8001"
go run main.go
Replica 2 (Port 8002):

$env:PORT="8002"
go run main.go
✅ After this, three backend servers will be running at:

http://localhost:8000

http://localhost:8001

http://localhost:8002

4. Start Frontend (Next.js)
Open a new terminal:
cd frontend/distributed-ui
npm install
npm run dev
✅ Frontend available at:
👉 http://localhost:3000


📋 Requirements
Golang (1.20+ recommended)

Node.js and npm (for frontend)