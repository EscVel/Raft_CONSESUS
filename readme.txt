first file
cmd /c 'curl.exe -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/join -d "{\"id\": \"node3\", \"addr\": \"127.0.0.1:7003\"}"'
./Raft_CONSENSUS -id node1 -raft-addr 127.0.0.1:7001 -http-addr 127.0.0.1:8001 -data-dir data -bootstrap

first run go build
./Raft_CONSENSUS -id node1 -raft-addr 127.0.0.1:7001 -http-addr 127.0.0.1:8001 -data-dir data -bootstrap
./Raft_CONSENSUS -id node2 -raft-addr 127.0.0.1:7002 -http-addr 127.0.0.1:8002 -data-dir data
./Raft_CONSENSUS -id node3 -raft-addr 127.0.0.1:7003 -http-addr 127.0.0.1:8003 -data-dir data

cmd /c 'curl.exe -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/join -d "{\"id\": \"node3\", \"addr\": \"127.0.0.1:7003\"}"'
cmd /c 'curl.exe -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/join -d "{\"id\": \"node2\", \"addr\": \"127.0.0.1:7002\"}"'

curl http://127.0.0.1:8002/status


Printers:
cmd /c 'curl.exe -L -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/printers -d "{\"id\": \"p2\", \"name\": \"testing\"}"'
cmd /c 'curl.exe http://127.0.0.1:8001/printers'


Fillaments:
cmd /c 'curl.exe -L -X POST -H "Content-Type: application/json" http://127.0.0.1:8003/filaments -d "{\"id\": \"f1\", \"type\": \"PLA\", \"color\": \"Blue\", \"weight_grams\": 1000}"'
cmd /c 'curl.exe http://127.0.0.1:8002/filaments'



testing:
go build

./Raft_CONSENSUS -id node1 -raft-addr 127.0.0.1:7001 -http-addr 127.0.0.1:8001 -data-dir data -bootstrap
./Raft_CONSENSUS -id node2 -raft-addr 127.0.0.1:7002 -http-addr 127.0.0.1:8002 -data-dir data
./Raft_CONSENSUS -id node3 -raft-addr 127.0.0.1:7003 -http-addr 127.0.0.1:8003 -data-dir data


cmd /c 'curl.exe -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/join -d "{\"id\": \"node2\", \"addr\": \"127.0.0.1:7002\"}"'
cmd /c 'curl.exe -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/join -d "{\"id\": \"node3\", \"addr\": \"127.0.0.1:7003\"}"'


Add a printer: 
cmd /c 'curl.exe -L -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/printers -d "{\"id\": \"p1\", \"name\": \"Ender 3 Pro\"}"'

List Printer:
cmd /c 'curl.exe http://127.0.0.1:8002/printers'

filaments
cmd /c 'curl.exe -L -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/filaments -d "{\"id\": \"f1\", \"type\": \"PLA\", \"color\": \"Blue\", \"weight_grams\": 1000}"'
cmd /c 'curl.exe http://127.0.0.1:8003/filaments'


cmd /c 'curl.exe -L -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/print_jobs -d "{\"id\": \"job1\", \"file_path\": \"/models/boat.gcode\", \"grams_needed\": 50, \"printer_id\": \"p1\", \"filament_id\": \"f1\"}"'
cmd /c 'curl.exe http://127.0.0.1:8002/print_jobs'
cmd /c 'curl.exe -L -X POST -H "Content-Type: application/json" http://127.0.0.1:8001/print_jobs -d "{\"id\": \"job2\", \"file_path\": \"/models/tower.gcode\", \"grams_needed\": 2000, \"printer_id\": \"p1\", \"filament_id\": \"f1\"}"'


cmd /c 'curl.exe -L -X POST http://127.0.0.1:8001/print_jobs/job1/status?status=Running'















dev notes:
tag:
    v1-stable = leader election working
