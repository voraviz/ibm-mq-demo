#!/bin/bash
QM1_ACTIVE_NODE=$(podman exec mq-node-1 dspmq -o nativeha -x | grep "ROLE(Active)" | grep -v QMNAME| awk '{print $3}'| sed -r 's/^[^(]*\(([^)]+)\).*/\1/')
QM2_ACTIVE_NODE=$(podman exec mq-node-4 dspmq -o nativeha -x | grep "ROLE(Active)" | grep -v QMNAME| awk '{print $3}'| sed -r 's/^[^(]*\(([^)]+)\).*/\1/')
while [ 1 ];
do
    clear
    QM1_CONN=$(podman exec $QM1_ACTIVE_NODE bash -c 'echo "dis conn(*) where(appltag eq '"'"'keshi'"'"') conntag" | runmqsc QM1' | grep APPLTAG | wc -l|sed 's/^[ \t]*//')
    QM2_CONN=$(podman exec $QM2_ACTIVE_NODE bash -c 'echo "dis conn(*) where(appltag eq '"'"'keshi'"'"') conntag" | runmqsc QM2' | grep APPLTAG | wc -l|sed 's/^[ \t]*//')
    echo "========================="
    podman exec $QM1_ACTIVE_NODE bash -c 'echo "DISPLAY APSTATUS('"'"'keshi'"'"') TYPE (APPL)" | runmqsc QM1'
    echo "========================="
    echo "QM1 Active on: $QM1_ACTIVE_NODE with $QM1_CONN connections"
    echo "QM2 Active on: $QM2_ACTIVE_NODE with $QM2_CONN connections"
    sleep 5
done