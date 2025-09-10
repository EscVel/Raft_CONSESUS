# Raft3D: A Fault-Tolerant 3D Print Management System

Raft3D is a backend application that simulates a management system for a 3D printing workshop. It uses the **Raft consensus algorithm** to provide a distributed, fault-tolerant data store instead of a traditional centralized database. This project demonstrates core distributed systems concepts like leader election, log replication, and fault tolerance through a practical RESTful API.

---
## Features

* **Distributed Consensus**: Utilizes HashiCorp's battle-tested Raft library to manage state changes.
* **Leader Election**: Automatically elects a new leader if the current one fails, ensuring the cluster remains available.
* **Fault Tolerance**: The 3-node cluster can tolerate the failure of a single node without any loss of data or availability.
* **Log Replication**: All state changes (like adding a printer or updating a job) are safely replicated to a majority of nodes before being applied.
* **Dynamic Membership**: Nodes can be dynamically added to the running cluster via a `/join` API endpoint.
* **Client Redirection**: Follower nodes automatically redirect write requests to the current leader, simplifying client-side logic.
* **RESTful API**: Provides a complete HTTP API to manage printers, filaments, and print jobs.
* **Business Logic Enforcement**: The state machine enforces critical rules, such as checking filament weight before starting a job and validating job status transitions.

---
## Prerequisites

To build and run this project, you will need:
* **Go** (version 1.18 or later)
* A command-line tool for making HTTP requests, such as **cURL**.

---
## Getting Started

Follow these steps to get your 3-node cluster up and running.

### 1. Build the Application
First, compile the source code into an executable.

```sh
go build
```

### 2. Launch the 3-Node Cluster
You will need to open **three separate terminals** or command prompts in your project directory. Run one command in each to start the nodes.

**Terminal 1 (Bootstrap Node):**
```powershell
./Raft_CONSENSUS -id node1 -raft-addr 127.0.0.1:7001 -http-addr 127.0.0.1:8001 -data-dir data -bootstrap
```

**Terminal 2 (Follower Node):**
```powershell
./Raft_CONSENSUS -id node2 -raft-addr 127.0.0.1:7002 -http-addr 127.0.0.1:8002 -data-dir data
```

**Terminal 3 (Follower Node):**
```powershell
./Raft_CONSENSUS -id node3 -raft-addr 127.0.0.1:7003 -http-addr 127.0.0.1:8003 -data-dir data
```
At this point, `node1` is the leader, but it does not know about the other two nodes.

### 3. Join the Follower Nodes
Open a **fourth terminal** to send API requests to the leader (`:8001`) to have the other nodes join the cluster.

**Join `node2`:**
```powershell
cmd /c 'curl.exe -X POST -H "Content-Type: application/json" [http://127.0.0.1:8001/join](http://127.0.0.1:8001/join) -d "{\"id\": \"node2\", \"addr\": \"127.0.0.1:7002\"}"'
```

**Join `node3`:**
```powershell
cmd /c 'curl.exe -X POST -H "Content-Type: application/json" [http://127.0.0.1:8001/join](http://127.0.0.1:8001/join) -d "{\"id\": \"node3\", \"addr\": \"127.0.0.1:7003\"}"'
```
Your cluster is now fully operational!

---
## API Endpoints

All `POST` requests that modify data should be sent with the `-L` flag in cURL to automatically follow redirects if the receiving node is not the leader.

| Method | Endpoint                                 | Description                                                                                              | Example `curl` Command                                                                                                                                                             |
| :---   | :---                                     | :---                                                                                                     | :---                                                                                                                                                                               |
| `POST` | `/join`                                  | (Internal) Joins a new node to the cluster.                                                              | `cmd /c 'curl.exe -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/join -d "{\"id\": \"node2\", \"addr\": \"127.0.0.1:7002\"}"'`                                    |
| `GET`  | `/status`                                | Gets the detailed Raft status of a specific node.                                                        | `cmd /c 'curl.exe http://127.0.0.1:8002/status'`                                                                                                                                   |
| `POST` | `/printers`                              | Creates a new printer.                                                                                   | `cmd /c 'curl.exe -L -X POST -H "Content-Type: application/json" http://127.0.0.1:8002/printers -d "{\"id\": \"p1\", \"name\": \"Ender 3 Pro\"}"'`                              |
| `GET`  | `/printers`                              | Lists all printers in the system.                                                                        | `cmd /c 'curl.exe http://127.0.0.1:8003/printers'`                                                                                                                                  |
| `POST` | `/filaments`                             | Creates a new filament spool.                                                                            | `cmd /c 'curl.exe -L -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/filaments -d "{\"id\": \"f1\", \"type\": \"PLA\", \"color\": \"Blue\", \"weight_grams\": 1000}"'` |
| `GET`  | `/filaments`                             | Lists all filament spools.                                                                               | `cmd /c 'curl.exe http://127.0.0.1:8001/filaments'`                                                                                                                                |
| `POST` | `/print_jobs`                            | Creates a new print job. Fails if filament is insufficient.                                              | `cmd /c 'curl.exe -L -X POST -H "Content-Type: application/json" http://127.0.0.1:8003/print_jobs -d "{\"id\": \"job1\", \"file_path\": \"/models/boat.gcode\", \"grams_needed\": 50, \"printer_id\": \"p1\", \"filament_id\": \"f1\"}"'` |
| `GET`  | `/print_jobs`                            | Lists all print jobs.                                                                                    | `cmd /c 'curl.exe http://127.0.0.1:8002/print_jobs'`                                                                                                                               |
| `POST` | `/print_jobs/{id}/status?status={state}` | Updates the status of a print job. Valid states: `Running`, `Done`, `Canceled`.                          | `cmd /c 'curl.exe -L -X POST http://127.0.0.1:8002/print_jobs/job1/status?status=Running'`                                                                                           |

---
## Technology Stack

* **Language**: Go
* **Consensus**: HashiCorp Raft
* **Storage**: HashiCorp raft-boltdb
