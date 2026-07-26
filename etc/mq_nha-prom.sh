#!/bin/bash
QMGR="${MQ_QMGR:-QM1}"
OUT_FILE="${MQ_TEXTFILE_PATH:-/var/lib/node_exporter/textfile_collector/mq_nha.prom}"
TMP_FILE="${OUT_FILE}.tmp"
#TMP_FILE="./test.tmp"
QMGR=QM1
raw=$(dspmq -o nativeha -x -m "$QMGR")

# Value out of a KEY(value) token, portable (no grep -P). The leading
# [^A-Za-z] guard stops ROLE/INSYNC from also matching GRPROLE/GRPINSYNC.
field() {
  printf '%s\n' "$2" | grep -oE "(^|[^A-Za-z])$1\([^)]*\)" | head -1 | sed -E 's/.*\(([^)]*)\)/\1/'
}

# Group-level quorum from the header line: QUORUM(insync/total)
quorum=$(echo "$raw" | head -1 | grep -oE 'QUORUM\([0-9]+/[0-9]+\)' | head -1)
insync=${quorum#QUORUM(}; insync=${insync%/*}
total=${quorum##*/};      total=${total%)}

{
  echo "# TYPE ibmmq_nha_quorum_insync gauge"
  echo "ibmmq_nha_quorum_insync{qmgr=\"$QMGR\"} $insync"
  echo "# TYPE ibmmq_nha_quorum_total gauge"
  echo "ibmmq_nha_quorum_total{qmgr=\"$QMGR\"} $total"
  echo "# TYPE ibmmq_nha_instance_role gauge"
  echo "# TYPE ibmmq_nha_instance_connected gauge"
  echo "# TYPE ibmmq_nha_instance_insync gauge"

  echo "$raw" | grep "INSTANCE(" | tail -n +1 | while IFS= read -r line; do
    inst=$(field INSTANCE "$line")
    role=$(field ROLE "$line")
    connactv=$(field CONNACTV "$line")
    insync_i=$(field INSYNC "$line")

    echo "ibmmq_nha_instance_role{qmgr=\"$QMGR\",instance=\"$inst\",role=\"$role\"} 1"
    [ "$connactv" = "yes" ] && cval=1 || cval=0
    echo "ibmmq_nha_instance_connected{qmgr=\"$QMGR\",instance=\"$inst\"} $cval"
    [ "$insync_i" = "yes" ] && ival=1 || ival=0
    echo "ibmmq_nha_instance_insync{qmgr=\"$QMGR\",instance=\"$inst\"} $ival"
  done
} > "$TMP_FILE"
mv "$TMP_FILE" "$OUT_FILE"