first file
cmd /c 'curl.exe -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/join -d "{\"id\": \"node3\", \"addr\": \"127.0.0.1:7003\"}"'
./Raft_CONSENSUS -id node1 -raft-addr 127.0.0.1:7001 -http-addr 127.0.0.1:8001 -data-dir data -bootstrap

first run go build
./Raft_CONSENSUS -id node1 -raft-addr 127.0.0.1:7001 -http-addr 127.0.0.1:8001 -data-dir data -bootstrap
./Raft_CONSENSUS -id node2 -raft-addr 127.0.0.1:7002 -http-addr 127.0.0.1:8002 -data-dir data
./Raft_CONSENSUS -id node3 -raft-addr 127.0.0.1:7003 -http-addr 127.0.0.1:8003 -data-dir data

cmd /c 'curl.exe -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/join -d "{\"id\": \"node3\", \"addr\": \"127.0.0.1:7003\"}"'
cmd /c 'curl.exe -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/join -d "{\"id\": \"node2\", \"addr\": \"127.0.0.1:7002\"}"'