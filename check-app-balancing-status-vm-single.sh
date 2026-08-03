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
    echo "=== Cluster summary ($APPLTAG) ==="
    # runmqsc emits multi-column KEY(VALUE) records; parse them into an aligned
    # table so the cluster-wide TYPE(APPL) roll-up and the per-QM TYPE(QMGR) rows
    # are readable at a glance. Records are separated by the AMQ8932I header line.
    printf '%s\n' "DISPLAY APSTATUS('$APPLTAG') TYPE(APPL)" "DISPLAY APSTATUS('$APPLTAG') TYPE(QMGR)" \
        | runmqsc "$QMGR" 2>/dev/null \
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
        echo "=== $QMGR local eligibility ==="
        echo "DISPLAY APSTATUS('$APPLTAG') TYPE(LOCAL) MOVABLE IMMREASN" \
            | runmqsc "$QMGR" 2>/dev/null \
            | grep -E "MOVABLE\(|IMMREASN\("
        echo ""
    fi
    echo "=== Connection counts ==="
    echo "$QMGR Active on: ${active:-<unavailable>} with $conn connections"
    sleep "$INTERVAL"
done
