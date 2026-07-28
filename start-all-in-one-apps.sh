#!/bin/bash
if [ ! -f "all-in-one-app/target/quarkus-app/quarkus-run.jar" ];
then
  echo "Compile all-in-one-app..."
  cd all-in-one-app;mvn clean package
  pwd
fi
cd all-in-one-app
QUARKUS_HTTP_PORT=9190
MAX_PORT=$(expr $QUARKUS_HTTP_PORT + 8 )

while [ $QUARKUS_HTTP_PORT -lt $MAX_PORT ]
do
    echo "Start all-in-one-app with port $QUARKUS_HTTP_PORT"
    rm -f ../logs/$QUARKUS_HTTP_PORT.log
    java -Dquarkus.http.port=$QUARKUS_HTTP_PORT \
    -Dibm.mq.ccdt-url="file:../ccdt/ccdt.cluster.json" \
    -Dibm.mq.application-name="jack" \
    -Dibm.mq.queue-manager="*UNIQA" \
    -Xmx512m \
    -jar target/quarkus-app/quarkus-run.jar 1>../logs/$QUARKUS_HTTP_PORT.log 2>&1 &
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