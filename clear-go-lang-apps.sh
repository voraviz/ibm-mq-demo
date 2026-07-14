#!/bin/sh
for i in $(ps -ef | grep api-app-go|grep -v grep | awk '{print $2}')
do
echo "Kill pid: $i"
kill -9 $i
done
rm -f logs/*.log
