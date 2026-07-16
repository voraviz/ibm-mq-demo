#!/bin/bash
mkdir -p logs
export SERVER_PORT=8100
MAX_PORT=$(expr $SERVER_PORT + 10 )
export MQAPPLNAME="keshi"
export IBM_MQ_QUEUE_MANAGER='*UNIQA'
export IBM_MQ_CCDT_URL='file:./ccdt/ccdt.cluster.json'

while [ $SERVER_PORT -lt $MAX_PORT ]
do
    echo "Start api-app-go on port $SERVER_PORT"
    rm -f $SERVER_PORT.log
    ./api-app-go/api-app-go 1> logs/$SERVER_PORT.log  2>&1 &
    sleep 5
    echo "Send message to localhost:$SERVER_PORT"
    COUNT=0
    while [ $COUNT -lt 10 ];
    do
        curl -X POST http://localhost:$SERVER_PORT/api/messages   -H "Content-Type: application/json"  -d '{"text":"test"}'
        COUNT=$(expr $COUNT + 1 )
    done
    echo "Start Listener..."
    curl -X POST http://localhost:$SERVER_PORT/api/consumer/start
    # echo "Get Connection..."
    # curl http://localhost:$SERVER_PORT/api/info|jq
    SERVER_PORT=$(expr $SERVER_PORT + 1 )
done
