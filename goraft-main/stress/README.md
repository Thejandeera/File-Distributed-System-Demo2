# Stress

The goal of this program is to automate basic correctness checks and
stress tests.

To run:

```console
$ cd cmd/stress
$ go run main.go util.go
```

With the `go-deadlock` package turned off and the default `sync`
package on, I get throughput of around 20k-40k entries/second with
this stress test.

## Background

This program runs a few servers in memory but still communicates over
localhost and actually writes/reads from disk.

The state machine tested is still a key-value state machine.

Most of the settings are configurable by editing
[./main.go](./main.go). But it's not particularly clean code.

## Tests

It attempts to do the following:

1. Start three servers and wait for a leader to be elected.
2. Insert `N_ENTRIES` in `BATCH_SIZE` across `N_CLIENTS`.
3. Validate that all servers have committed all messages they are
   aware of.
4. Validate that all servers have all entries in their log in the
   correct order that entries were inserted.
5. Shut down all servers and turn them back on. Validate that a leader has been elected.
6. Validate that all messages that were inserted before shutdown are
   committed and in the log in the correct order.
7. Shut down all servers and delete the log for one server.
8. Turn all servers back on.
9. Validate that a leader has been elected.
10. Ensure that all servers have all entries (i.e. that the deleted log has been recovered).

That is: test the basics of leader election and log replication.

One variation that I run manually at the moment is to have three
servers configured but only turn on two of them and ensure the entire
process still works (ignoring testing for entries on the down
server). This is to prove that quorum consensus works for leader
election and log replication.

## From Stress Test to Distributed File System

This Raft implementation can serve as the core consistency engine for a distributed file system. The key is to replace the simple `kvStateMachine` used in this test with a more sophisticated `fileSystemStateMachine`.

### 1. The Role of Raft: Managing Metadata

In a distributed file system, Raft's primary role is to manage the **metadata**, not the actual file data. This is the "control plane" of your system. The metadata includes:

*   The directory tree structure (e.g., which files are in which directories).
*   File attributes like permissions, size, and modification times.
*   A mapping of file chunks to the data nodes that store them.

### 2. Building a `fileSystemStateMachine`

Your new state machine's `Apply` method would need to understand file system operations. Instead of `set` and `get`, the commands in your Raft log would be operations like:

*   `CreateFile(path, attributes)`
*   `Mkdir(path)`
*   `WriteChunk(file_id, chunk_index, datanode_address)`
*   `Delete(path)`

When a client wants to create a directory, it sends a `Mkdir` command to the Raft cluster. Raft ensures this command is replicated and applied to every server's `fileSystemStateMachine` in the exact same order, keeping every node's view of the file system structure identical.

### 3. Important Next Steps for a Real System

This educational project is a great start, but a production-grade file system would require implementing features mentioned in the main `README.md`:

*   **Snapshotting (Most Critical):** The log of all file system operations will grow very large. You must implement snapshotting to periodically save the entire metadata state to a file and truncate the log. Without this, server restarts would become impossibly slow.
*   **Cluster Membership Changes:** To add or remove servers from your cluster without downtime, you need to implement Raft's protocol for dynamic membership changes.
