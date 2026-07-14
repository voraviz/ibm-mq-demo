#!/bin/bash
for i in 1 2 3 4 5 6;
do
 podman stop mq-node-$i
 podman rm -f mq-node-$i
done
for i in mq-node-1-data mq-node-2-data mq-node-3-data mq-node-4-data mq-node-5-data mq-node-6-data; do
  podman volume exists $i && podman volume rm $i
  podman volume create $i
done
TAG=10.0.0.0-r1-amd64
MQ_PORT=1414
MQ_PROMETHEUS_PORT=9517
MQ_CONSOLE_PORT=9443
QUEUE_MANAGER=QM1
QUEUE_MANAGER_LOWER=qm1
for i in 1 2 3 4 5 6;
do
  if [ $i -gt 3 ];
  then
    QUEUE_MANAGER=QM2
  fi
  echo "Creating Node $i ......"
  podman run -d \
  --secret mqAdminPassword --secret mqAppPassword \
  --name mq-node-$i \
  --platform linux/amd64 \
  --network mq-ha-net \
  --hostname mq-node-$i \
  --cpus 1 --memory 800m \
  -p $MQ_PORT:1414 \
  -p $MQ_CONSOLE_PORT:9443 \
  -p $MQ_PROMETHEUS_PORT:9517 \
  -v mq-node-$i-data:/var/mqm \
  -v ./mq-native-ha/config/qm-node$i-cluster.ini:/etc/mqm/native-ha.ini:ro \
  -v ./mq-native-ha/config/config.cluster.mqsc:/etc/mqm/config.mqsc:ro \
  -e LICENSE=accept \
  -e MQ_QMGR_NAME=$QUEUE_MANAGER \
  -e MQ_NATIVE_HA=true \
  -e MQ_ENABLE_EMBEDDED_WEB_SERVER=true \
  -e MQ_ENABLE_METRICS=true \
  icr.io/ibm-messaging/mq:$TAG
  MQ_PORT=$(expr $MQ_PORT + 1 )
  MQ_PROMETHEUS_PORT=$(expr $MQ_PROMETHEUS_PORT + 1 )
  MQ_CONSOLE_PORT=$(expr $MQ_CONSOLE_PORT + 1 )
done
sleep 60
clear
podman ps --format "table {{.Names}}\t{{.Ports}}\t{{.Status}}"
sleep 10
clear
echo "Wait for QM1 Native HA setting..."
sleep 60
QM1_ACTIVE_NODE=$(podman exec mq-node-1 dspmq -o nativeha -x | grep "ROLE(Active)" | grep -v QMNAME| awk '{print $3}'| sed -r 's/^[^(]*\(([^)]+)\).*/\1/')
echo "QM1 Active on: $QM1_ACTIVE_NODE"
clear
echo "Wait for QM2 Native HA setting..."
sleep 60
QM2_ACTIVE_NODE=$(podman exec mq-node-4 dspmq -o nativeha -x | grep "ROLE(Active)" | grep -v QMNAME| awk '{print $3}'| sed -r 's/^[^(]*\(([^)]+)\).*/\1/')
echo "QM2 Active on: $QM1_ACTIVE_NODE"
sleep 10
clear
echo "Wait for cluster setup..."
sleep 60
podman exec mq-node-1 bash -c "echo 'display clusqmgr(*)' | runmqsc QM1"
sleep 60
podman exec mq-node-4 bash -c "echo 'display clusqmgr(*)' | runmqsc QM2"
