#!/bin/bash
QUARKUS_HTTP_PORT=9190
MAX_PORT=$(expr $QUARKUS_HTTP_PORT + 10 )


while [ $QUARKUS_HTTP_PORT -lt $MAX_PORT ]
do
    echo "Start all-in-one-app with port $QUARKUS_HTTP_PORT"
    podman run  --name all-in-one-$QUARKUS_HTTP_PORT \
           --detach \
           --memory 500m \
           -v ./ccdt/ccdt.cluster.container.json:/config/ccdt.json:ro \
           -e IBM_MQ_CCDT_URL="file:///config/ccdt.json" \
           -e IBM_MQ_APPLICATION_NAME="jack" \
           -e IBM_MQ_QUEUE_MANAGER="*UNIQA" \
           -p $QUARKUS_HTTP_PORT:8080 \
           quay.io/voravitl/simple-mq-app:latest
    set +x
    sleep 5
    echo "Send message to localhost:$QUARKUS_HTTP_PORT"
    COUNT=0
    while [ $COUNT -lt 10 ];
    do
        curl -X POST http://localhost:$QUARKUS_HTTP_PORT/api/messages  \
         -H "Content-Type: application/json"  \
         -d '{"text":"test"}'
        COUNT=$(expr $COUNT + 1 )
        echo "Message no. $COUNT"
    done
    echo "Start Listener..."
    curl -X POST http://localhost:$QUARKUS_HTTP_PORT/api/consumer/start
    QUARKUS_HTTP_PORT=$(expr $QUARKUS_HTTP_PORT + 1 )
done