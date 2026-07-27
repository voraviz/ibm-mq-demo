#!/bin/bash
# Original: https://github.com/ibm-messaging/mq-uniform-clusters/blob/master/demo/M1MacDocker/connections.sh
# Usage: ./check-app-balancing-status.sh <appltag>
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

if [[ -z "$1" ]]; then
    echo "Usage: $0 <appltag>" >&2
    exit 1
fi

# The application tag to monitor, supplied as the first argument.
APPLTAG="$1"

# Helper: resolve which NativeHA node is currently Active for a given seed node.
get_active_node() {
    local seed_node="$1"
    "$CONTAINER_ENGINE" exec "$seed_node" dspmq -o nativeha -x \
        | grep "ROLE(Active)" \
        | grep -v QMNAME \
        | awk '{print $3}' \
        | sed -r 's/^[^(]*\(([^)]+)\).*/\1/'
}

# Helper: count APPLTAG connections on a queue manager.
count_connections() {
    local node="$1"
    local qmgr="$2"
    "$CONTAINER_ENGINE" exec "$node" \
        bash -c "echo \"dis conn(*) where(appltag eq '$APPLTAG') conntag\" | runmqsc $qmgr" \
        | grep -c "APPLTAG"
}

while true; do
    # Refresh active-node resolution each iteration so failovers are detected.
    QM1_ACTIVE_NODE=$(get_active_node mq-node-1)
    QM2_ACTIVE_NODE=$(get_active_node mq-node-4)

    QM1_CONN=$(count_connections "$QM1_ACTIVE_NODE" QM1)
    QM2_CONN=$(count_connections "$QM2_ACTIVE_NODE" QM2)

    clear
    echo "=== Cluster summary ==="
    "$CONTAINER_ENGINE" exec "$QM1_ACTIVE_NODE" \
        bash -c "printf '%s\n' \"DISPLAY APSTATUS('$APPLTAG') TYPE(APPL)\" \"DISPLAY APSTATUS('$APPLTAG') TYPE(QMGR)\" | runmqsc QM1"
    echo "=== QM1 local eligibility ==="
    "$CONTAINER_ENGINE" exec "$QM1_ACTIVE_NODE" \
        bash -c "echo \"DISPLAY APSTATUS('$APPLTAG') TYPE(LOCAL) ALL\" | runmqsc QM1"
    echo "=== QM2 local eligibility ==="
    "$CONTAINER_ENGINE" exec "$QM2_ACTIVE_NODE" \
        bash -c "echo \"DISPLAY APSTATUS('$APPLTAG') TYPE(LOCAL) ALL\" | runmqsc QM2"
    echo "=== Connection counts ==="
    echo "QM1 Active on: $QM1_ACTIVE_NODE with $QM1_CONN connections"
    echo "QM2 Active on: $QM2_ACTIVE_NODE with $QM2_CONN connections"
    sleep 5
done
