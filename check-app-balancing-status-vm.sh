#!/bin/bash
# Monitor uniform-cluster app balancing for IBM MQ running natively on a VM.
#
# Run this ON the MQ VM: it uses local `runmqsc`/`dspmq` (bindings mode) — no
# podman, no ssh. It is the VM counterpart of check-app-balancing-status.sh
# (containers) and of the original client-based script:
#   https://github.com/ibm-messaging/mq-uniform-clusters/blob/master/demo/M1MacDocker/connections.sh
#
# Native HA note: a queue manager only accepts local connections on its Active
# instance, so `runmqsc QM1` here talks to whichever instance is Active — no
# need to resolve and hop into a node as the container version does.
#
# Usage:   ./check-app-balancing-status-vm.sh <appltag> [qmgr ...]
# Example: ./check-app-balancing-status-vm.sh keshi QM1 QM2
# Env:     INTERVAL=<seconds>  (default 5)

set -u

APPLTAG="${1:-}"
if [[ -z "$APPLTAG" ]]; then
    echo "Usage: $0 <appltag> [qmgr ...]" >&2
    exit 1
fi
shift

# Queue managers to watch — default to the demo's two HA groups.
QMGRS=("$@")
[[ ${#QMGRS[@]} -eq 0 ]] && QMGRS=(QM1 QM2)

INTERVAL="${INTERVAL:-5}"

# Resolve the Active Native HA instance name for a QM (empty if unavailable).
# Same field/parse layout as the container script's dspmq pipeline, filtered to
# one QM with -m instead of running inside a specific node's container.
get_active_instance() {
    local qmgr="$1"
    dspmq -m "$qmgr" -o nativeha -x 2>/dev/null \
        | grep "ROLE(Active)" \
        | grep -v QMNAME \
        | awk '{print $3}' \
        | sed -r 's/^[^(]*\(([^)]+)\).*/\1/'
}

# Count APPLTAG connections on a QM via local runmqsc.
count_connections() {
    local qmgr="$1"
    echo "dis conn(*) where(appltag eq '$APPLTAG') conntag" \
        | runmqsc "$qmgr" 2>/dev/null \
        | grep -c "APPLTAG"
}

while true; do
    clear
    for qmgr in "${QMGRS[@]}"; do
        active=$(get_active_instance "$qmgr")
        conn=$(count_connections "$qmgr")
        echo "========================="
        echo "DISPLAY APSTATUS('$APPLTAG') TYPE(APPL)" | runmqsc "$qmgr" 2>/dev/null
        echo "========================="
        echo "$qmgr Active on: ${active:-<unavailable>} with $conn connections"
    done
    sleep "$INTERVAL"
done
