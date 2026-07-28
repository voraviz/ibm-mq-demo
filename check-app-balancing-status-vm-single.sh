#!/bin/bash
# Monitor uniform-cluster app balancing for ONE queue manager, on its own VM.
#
# Use this when ssh between VMs is not allowed: run it on each MQ VM for that
# VM's local queue manager. It uses local `runmqsc`/`dspmq` (bindings mode).
# It is the single-QM counterpart of check-app-balancing-status-vm.sh.
#
# Usage:   ./check-app-balancing-status-vm-single.sh <qmgr> <appltag>
# Example: ./check-app-balancing-status-vm-single.sh QM1 keshi
# Env:     INTERVAL=<seconds>    (default 5)
#          SHOW_LOCAL=1          also print per-instance MOVABLE / IMMREASN

set -u

QMGR="${1:-}"
APPLTAG="${2:-}"
if [[ -z "$QMGR" || -z "$APPLTAG" ]]; then
    echo "Usage: $0 <qmgr> <appltag>" >&2
    exit 1
fi

INTERVAL="${INTERVAL:-5}"
SHOW_LOCAL="${SHOW_LOCAL:-0}"

# Resolve the Active Native HA instance name (empty if unavailable).
get_active_instance() {
    dspmq -m "$QMGR" -o nativeha -x 2>/dev/null \
        | grep "ROLE(Active)" \
        | grep -v QMNAME \
        | awk '{print $3}' \
        | sed -r 's/^[^(]*\(([^)]+)\).*/\1/'
}

# Count APPLTAG connections on the queue manager via local runmqsc.
count_connections() {
    echo "dis conn(*) where(appltag eq '$APPLTAG') conntag" \
        | runmqsc "$QMGR" 2>/dev/null \
        | grep -c "APPLTAG"
}

while true; do
    active=$(get_active_instance)
    conn=$(count_connections)
    clear
    echo "========================="
    echo "DISPLAY APSTATUS('$APPLTAG') TYPE(APPL)" | runmqsc "$QMGR" 2>/dev/null
    if [ "$SHOW_LOCAL" = "1" ]; then
        echo "--- $QMGR local eligibility ---"
        echo "DISPLAY APSTATUS('$APPLTAG') TYPE(LOCAL) MOVABLE IMMREASN" \
            | runmqsc "$QMGR" 2>/dev/null \
            | grep -E "MOVABLE\(|IMMREASN\("
    fi
    echo "========================="
    echo "$QMGR Active on: ${active:-<unavailable>} with $conn connections"
    sleep "$INTERVAL"
done
