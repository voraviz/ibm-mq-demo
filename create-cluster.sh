#!/bin/bash
set -e

if [ -n "${CONTAINER_ENGINE:-}" ]; then
  if ! command -v "$CONTAINER_ENGINE" >/dev/null 2>&1 || ! "$CONTAINER_ENGINE" info >/dev/null 2>&1; then
    echo "CONTAINER_ENGINE '$CONTAINER_ENGINE' is not available or its engine is not running." >&2
    exit 1
  fi
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

echo "Using container engine: $CONTAINER_ENGINE"

if [ "$CONTAINER_ENGINE" = "docker" ]; then
  MQ_ADMIN_PASSWORD=${MQ_ADMIN_PASSWORD:-passw0rd}
  MQ_APP_PASSWORD=${MQ_APP_PASSWORD:-passw0rd}
  MQ_CREDENTIAL_ARGS=(
    -e "MQ_ADMIN_PASSWORD=$MQ_ADMIN_PASSWORD"
    -e "MQ_APP_PASSWORD=$MQ_APP_PASSWORD"
  )
else
  MQ_CREDENTIAL_ARGS=(--secret mqAdminPassword --secret mqAppPassword)
fi

for i in 1 2 3 4 5 6;
do
 "$CONTAINER_ENGINE" rm -f mq-node-$i 2>/dev/null || true
done
for e in mq-exporter-qm1 mq-exporter-qm2;
do
 "$CONTAINER_ENGINE" rm -f "$e" 2>/dev/null || true
done
for i in mq-node-1-data mq-node-2-data mq-node-3-data mq-node-4-data mq-node-5-data mq-node-6-data; do
  "$CONTAINER_ENGINE" volume inspect "$i" >/dev/null 2>&1 && "$CONTAINER_ENGINE" volume rm "$i"
  "$CONTAINER_ENGINE" volume create "$i"
done

"$CONTAINER_ENGINE" network inspect mq-ha-net >/dev/null 2>&1 ||
  "$CONTAINER_ENGINE" network create mq-ha-net >/dev/null

TAG=10.0.0.0-r1-amd64
MQ_PORT=1414
MQ_PROMETHEUS_PORT=9157
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
  "$CONTAINER_ENGINE" run -d \
  "${MQ_CREDENTIAL_ARGS[@]}" \
  --name mq-node-$i \
  --platform linux/amd64 \
  --network mq-ha-net \
  --hostname mq-node-$i \
  --cpus 1 --memory 800m \
  -p $MQ_PORT:1414 \
  -p $MQ_CONSOLE_PORT:9443 \
  -p $MQ_PROMETHEUS_PORT:9157 \
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
clear 2>/dev/null || true
"$CONTAINER_ENGINE" ps --format "table {{.Names}}\t{{.Ports}}\t{{.Status}}"
sleep 10
clear 2>/dev/null || true
echo "Wait for QM1 Native HA setting..."
sleep 60
QM1_ACTIVE_NODE=$("$CONTAINER_ENGINE" exec mq-node-1 dspmq -o nativeha -x | grep "ROLE(Active)" | grep -v QMNAME| awk '{print $3}'| sed -r 's/^[^(]*\(([^)]+)\).*/\1/')
echo "QM1 Active on: $QM1_ACTIVE_NODE"
clear 2>/dev/null || true
echo "Wait for QM2 Native HA setting..."
sleep 60
QM2_ACTIVE_NODE=$("$CONTAINER_ENGINE" exec mq-node-4 dspmq -o nativeha -x | grep "ROLE(Active)" | grep -v QMNAME| awk '{print $3}'| sed -r 's/^[^(]*\(([^)]+)\).*/\1/')
echo "QM2 Active on: $QM2_ACTIVE_NODE"
sleep 10
clear 2>/dev/null || true
echo "Wait for cluster setup..."
sleep 60
"$CONTAINER_ENGINE" exec mq-node-1 bash -c "echo 'display clusqmgr(*)' | runmqsc QM1"
sleep 60
"$CONTAINER_ENGINE" exec mq-node-4 bash -c "echo 'display clusqmgr(*)' | runmqsc QM2"

# Per-queue metrics (e.g. ibmmq_queue_depth) — one exporter per HA group, each
# following its own active node. Built once with etc/build-mq-exporter.sh.
#EXPORTER_IMAGE=quay.io/voravitl/mq-prometheus:latest
#echo "Starting MQ metrics exporters — QM1 on :9257, QM2 on :9258 ..."
#"$CONTAINER_ENGINE" run --platform=linux/amd64 -d \
#  --name mq-exporter-qm1 \
#  --network mq-ha-net \
#  -p 9257:9157 \
#  -v ./mq-native-ha/config/mq_prometheus.qm1.yaml:/opt/config/mq_prometheus.yaml:ro \
#  "$EXPORTER_IMAGE"
#"$CONTAINER_ENGINE" run --platform=linux/amd64 -d \
#  --name mq-exporter-qm2 \
#  --network mq-ha-net \
#  -p 9258:9157 \
#  -v ./mq-native-ha/config/mq_prometheus.qm2.yaml:/opt/config/mq_prometheus.yaml:ro \
#  "$EXPORTER_IMAGE"
