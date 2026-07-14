#!/bin/bash
podman stop -a;podman rm -a -f
for i in mq-node-1-data mq-node-2-data mq-node-3-data; do
  podman volume exists $i && podman volume rm $i
  podman volume create $i
done
TAG=10.0.0.0-r1-amd64
CONFIG=config.auth.mqsc
MQ_PORT=1414
MQ_PROMETHEUS_PORT=9517
MQ_CONSOLE_PORT=9443
for i in 1 2 3;
do
  echo "Creating Node $i ......"
  podman run -d \
    --secret mqAdminPassword --secret mqAppPassword \
    --name mq-node-$i \
    --platform linux/amd64 \
    --network mq-ha-net \
    --hostname mq-node-$i \
    -p $MQ_PORT:1414 \
    -p $MQ_CONSOLE_PORT:9443 \
    -p $MQ_PROMETHEUS_PORT:9517 \
    -v mq-node-$i-data:/var/mqm \
    -v ./mq-native-ha/config/qm-node$i.ini:/etc/mqm/native-ha.ini:ro \
    -v ./mq-native-ha/config/$CONFIG:/etc/mqm/config.mqsc:ro \
    -e LICENSE=accept \
    -e MQ_QMGR_NAME=QM1 \
    -e MQ_NATIVE_HA=true \
    -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
    -e MQ_ENABLE_METRICS=true \
    icr.io/ibm-messaging/mq:$TAG
    MQ_PORT=$(expr $MQ_PORT + 1 )
    MQ_PROMETHEUS_PORT=$(expr $MQ_PROMETHEUS_PORT + 1 )
    MQ_CONSOLE_PORT=$(expr $MQ_CONSOLE_PORT + 1 )
done
sleep 10
podman ps --format "table {{.Names}}\t{{.Ports}}\t{{.Status}}"
sleep 10
clear
echo "Wait for QM1 Native HA setting..."
sleep 60
QM1_ACTIVE_NODE=$(podman exec mq-node-1 dspmq -o nativeha -x | grep "ROLE(Active)" | grep -v QMNAME| awk '{print $3}'| sed -r 's/^[^(]*\(([^)]+)\).*/\1/')
echo "QM1 Active on: $QM1_ACTIVE_NODE"
# ── Node 1 ──────────────────────────────────────────────────────────────────
# podman run -d \
#   --secret mqAdminPassword --secret mqAppPassword \
#   --name mq-node-1 \
#   --platform linux/amd64 \
#   --network mq-ha-net \
#   --hostname mq-node-1 \
#   -p 1414:1414 \
#   -p 9443:9443 \
#   -p 9157:9157 \
#   -v mq-node-1-data:/var/mqm \
#   -v ./mq-native-ha/config/qm-node1.ini:/etc/mqm/native-ha.ini:ro \
#   -v ./mq-native-ha/config/$CONFIG:/etc/mqm/config.mqsc:ro \
#   -e LICENSE=accept \
#   -e MQ_QMGR_NAME=QM1 \
#   -e MQ_NATIVE_HA=true \
#   -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
#   -e MQ_ENABLE_METRICS=true \
#   icr.io/ibm-messaging/mq:$TAG

# # ── Node 2 ──────────────────────────────────────────────────────────────────
# podman run -d \
#   --secret mqAdminPassword --secret mqAppPassword \
#   --name mq-node-2 \
#   --platform linux/amd64 \
#   --network mq-ha-net \
#   --hostname mq-node-2 \
#   -p 1415:1414 \
#   -p 9444:9443 \
#   -p 9158:9157 \
#   -v mq-node-2-data:/var/mqm \
#   -v ./mq-native-ha/config/qm-node2.ini:/etc/mqm/native-ha.ini:ro \
#   -v ./mq-native-ha/config/$CONFIG:/etc/mqm/config.mqsc:ro \
#   -e LICENSE=accept \
#   -e MQ_QMGR_NAME=QM1 \
#   -e MQ_NATIVE_HA=true \
#   -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
#   -e MQ_ENABLE_METRICS=true \
#   icr.io/ibm-messaging/mq:$TAG

# # ── Node 3 ──────────────────────────────────────────────────────────────────
# podman run -d \
#   --secret mqAdminPassword --secret mqAppPassword \
#   --name mq-node-3 \
#   --platform linux/amd64 \
#   --network mq-ha-net \
#   --hostname mq-node-3 \
#   -p 1416:1414 \
#   -p 9445:9443 \
#   -p 9159:9157 \
#   -v mq-node-3-data:/var/mqm \
#   -v ./mq-native-ha/config/qm-node3.ini:/etc/mqm/native-ha.ini:ro \
#   -v ./mq-native-ha/config/$CONFIG:/etc/mqm/config.mqsc:ro \
#   -e LICENSE=accept \
#   -e MQ_QMGR_NAME=QM1 \
#   -e MQ_NATIVE_HA=true \
#   -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
#   -e MQ_ENABLE_METRICS=true \
#   icr.io/ibm-messaging/mq:$TAG