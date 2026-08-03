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
# Env:     INTERVAL=<seconds>    (default 5)
#          SHOW_LOCAL=1          also print per-instance MOVABLE / IMMREASN

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
SHOW_LOCAL="${SHOW_LOCAL:-0}"

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
    # APSTATUS TYPE(APPL)/TYPE(QMGR) is a cluster-wide view — query it once from
    # the first QM (not per-QM, which would just repeat the same roll-up).
    echo "=== Cluster summary ($APPLTAG) ==="
    # runmqsc emits multi-column KEY(VALUE) records; parse them into an aligned
    # table so the cluster-wide TYPE(APPL) roll-up and the per-QM TYPE(QMGR) rows
    # are readable at a glance. Records are separated by the AMQ8932I header line.
    printf '%s\n' "DISPLAY APSTATUS('$APPLTAG') TYPE(APPL)" "DISPLAY APSTATUS('$APPLTAG') TYPE(QMGR)" \
        | runmqsc "${QMGRS[0]}" 2>/dev/null \
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

    for qmgr in "${QMGRS[@]}"; do
        active=$(get_active_instance "$qmgr")
        conn=$(count_connections "$qmgr")
        if [ "$SHOW_LOCAL" = "1" ]; then
            echo "=== $qmgr local eligibility ==="
            echo "DISPLAY APSTATUS('$APPLTAG') TYPE(LOCAL) MOVABLE IMMREASN" \
                | runmqsc "$qmgr" 2>/dev/null \
                | grep -E "MOVABLE\(|IMMREASN\("
            echo ""
        fi
        echo "$qmgr Active on: ${active:-<unavailable>} with $conn connections"
    done
    sleep "$INTERVAL"
done
