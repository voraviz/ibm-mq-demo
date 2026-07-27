#!/bin/bash
set -e

if [ -n "${CONTAINER_ENGINE:-}" ]; then
    "$CONTAINER_ENGINE" info >/dev/null
else
    for candidate in docker podman; do
        if command -v "$candidate" >/dev/null 2>&1 && "$candidate" info >/dev/null 2>&1; then
            CONTAINER_ENGINE="$candidate"
            break
        fi
    done
fi

if [ -z "${CONTAINER_ENGINE:-}" ]; then
    echo "A running Docker or Podman engine is required." >&2
    exit 1
fi

if [ "$CONTAINER_ENGINE" = "docker" ]; then
    CCDT_FILE=${CCDT_FILE:-./ccdt/ccdt.cluster.docker.json}
else
    CCDT_FILE=${CCDT_FILE:-./ccdt/ccdt.cluster.container.json}
fi

QUARKUS_HTTP_PORT=9190
APP_INSTANCE_COUNT=${APP_INSTANCE_COUNT:-10}
MAX_PORT=$(expr "$QUARKUS_HTTP_PORT" + "$APP_INSTANCE_COUNT")


while [ $QUARKUS_HTTP_PORT -lt $MAX_PORT ]
do
    echo "Start all-in-one-app with port $QUARKUS_HTTP_PORT"
    "$CONTAINER_ENGINE" run --name all-in-one-$QUARKUS_HTTP_PORT \
           --detach \
           --memory 500m \
           -v "$CCDT_FILE":/config/ccdt.json:ro \
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
