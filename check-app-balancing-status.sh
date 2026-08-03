#!/bin/bash
# Original: https://github.com/ibm-messaging/mq-uniform-clusters/blob/master/demo/M1MacDocker/connections.sh
# Usage: ./check-app-balancing-status.sh <appltag>
# Set SHOW_LOCAL=1 to include per-instance MOVABLE and IMMREASN details.
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
SHOW_LOCAL=${SHOW_LOCAL:-0}

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
    echo "=== Cluster summary ($APPLTAG) ==="
    # runmqsc emits multi-column KEY(VALUE) records; parse them into an aligned
    # table so the cluster-wide TYPE(APPL) roll-up and the per-QM TYPE(QMGR) rows
    # are readable at a glance. Records are separated by the AMQ8932I header line.
    "$CONTAINER_ENGINE" exec "$QM1_ACTIVE_NODE" \
        bash -c "printf '%s\n' \"DISPLAY APSTATUS('$APPLTAG') TYPE(APPL)\" \"DISPLAY APSTATUS('$APPLTAG') TYPE(QMGR)\" | runmqsc QM1" \
        | awk '
            /APSTATUS/ { next }            # skip echoed command lines
            /AMQ8932I/ { flush(); next }   # record boundary
            {
                s = $0
                while (match(s, /[A-Z0-9_]+\([^)]*\)/)) {
                    tok = substr(s, RSTART, RLENGTH)
                    key = tok; sub(/\(.*/, "", key)
                    val = substr(tok, index(tok, "(") + 1, length(tok) - index(tok, "(") - 1)
                    cur[key] = val
                    s = substr(s, RSTART + RLENGTH)
                }
            }
            function flush(   k) {
                if (cur["APPLNAME"] == "") { for (k in cur) delete cur[k]; return }
                if (cur["TYPE"] == "APPL") {
                    a_cl = cur["CLUSTER"]; a_ct = cur["COUNT"]; a_mv = cur["MOVCOUNT"]; a_bal = cur["BALANCED"]; have_a = 1
                } else {
                    n++
                    q_nm[n] = cur["QMNAME"]; q_ct[n] = cur["COUNT"]; q_mv[n] = cur["MOVCOUNT"]
                    q_st[n] = cur["BALSTATE"]; q_ac[n] = cur["ACTIVE"]
                    q_dt[n] = cur["LMSGDATE"]; q_tm[n] = cur["LMSGTIME"]
                }
                for (k in cur) delete cur[k]
            }
            END {
                flush()
                if (have_a)
                    printf "Cluster %-6s  count=%-3s movcount=%-3s balanced=%s\n\n", a_cl, a_ct, a_mv, a_bal
                printf "%-5s %-6s %-8s %-9s %-7s %s\n", "QMgr", "Count", "MovCount", "BalState", "Active", "LastMsg"
                printf "%-5s %-6s %-8s %-9s %-7s %s\n", "----", "-----", "--------", "--------", "------", "-------------------"
                for (i = 1; i <= n; i++)
                    printf "%-5s %-6s %-8s %-9s %-7s %s %s\n", q_nm[i], q_ct[i], q_mv[i], q_st[i], q_ac[i], q_dt[i], q_tm[i]
            }
        '
    echo ""

    if [ "$SHOW_LOCAL" = "1" ]; then
        echo "=== QM1 local eligibility ==="
        "$CONTAINER_ENGINE" exec "$QM1_ACTIVE_NODE" \
            bash -c "echo \"DISPLAY APSTATUS('$APPLTAG') TYPE(LOCAL) MOVABLE IMMREASN\" | runmqsc QM1" \
            | grep -E "MOVABLE\\(|IMMREASN\\("
        echo ""
        echo "=== QM2 local eligibility ==="
        "$CONTAINER_ENGINE" exec "$QM2_ACTIVE_NODE" \
            bash -c "echo \"DISPLAY APSTATUS('$APPLTAG') TYPE(LOCAL) MOVABLE IMMREASN\" | runmqsc QM2" \
            | grep -E "MOVABLE\\(|IMMREASN\\("
        echo ""
    fi

    echo "=== Connection counts ==="
    echo "QM1 Active on: $QM1_ACTIVE_NODE with $QM1_CONN connections"
    echo "QM2 Active on: $QM2_ACTIVE_NODE with $QM2_CONN connections"
    echo "Set SHOW_LOCAL=1 to display MOVABLE and IMMREASN for each instance."
    sleep 5
done
