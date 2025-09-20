This is a [Next.js](https://nextjs.org) project bootstrapped with [`create-next-app`](https://nextjs.org/docs/app/api-reference/cli/create-next-app).

## Getting Started

First, run the development server:

```bash
npm run dev
# or
yarn dev
# or
pnpm dev
# or
bun dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

You can start editing the page by modifying `app/page.tsx`. The page auto-updates as you edit the file.

This project uses [`next/font`](https://nextjs.org/docs/app/building-your-application/optimizing/fonts) to automatically optimize and load [Geist](https://vercel.com/font), a new font family for Vercel.

## Learn More

To learn more about Next.js, take a look at the following resources:

- [Next.js Documentation](https://nextjs.org/docs) - learn about Next.js features and API.
- [Learn Next.js](https://nextjs.org/learn) - an interactive Next.js tutorial.

You can check out [the Next.js GitHub repository](https://github.com/vercel/next.js) - your feedback and contributions are welcome!

## Deploy on Vercel

The easiest way to deploy your Next.js app is to use the [Vercel Platform](https://vercel.com/new?utm_medium=default-template&filter=next.js&utm_source=create-next-app&utm_campaign=create-next-app-readme) from the creators of Next.js.

Check out our [Next.js deployment documentation](https://nextjs.org/docs/app/building-your-application/deploying) for more details.



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