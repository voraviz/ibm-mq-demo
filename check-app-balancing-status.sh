#!/bin/bash

# Usage: ./check-app-balancing-status.sh <appltag>
if [[ -z "$1" ]]; then
    echo "Usage: $0 <appltag>" >&2
    exit 1
fi

# The application tag to monitor, supplied as the first argument.
APPLTAG="$1"

# Helper: resolve which NativeHA node is currently Active for a given seed node.
get_active_node() {
    local seed_node="$1"
    podman exec "$seed_node" dspmq -o nativeha -x \
        | grep "ROLE(Active)" \
        | grep -v QMNAME \
        | awk '{print $3}' \
        | sed -r 's/^[^(]*\(([^)]+)\).*/\1/'
}

# Helper: count APPLTAG connections on a queue manager.
count_connections() {
    local node="$1"
    local qmgr="$2"
    podman exec "$node" \
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
    echo "========================="
    podman exec "$QM1_ACTIVE_NODE" \
        bash -c "echo \"DISPLAY APSTATUS('$APPLTAG') TYPE(APPL)\" | runmqsc QM1"
    echo "========================="
    echo "QM1 Active on: $QM1_ACTIVE_NODE with $QM1_CONN connections"
    echo "QM2 Active on: $QM2_ACTIVE_NODE with $QM2_CONN connections"
    sleep 5
done