#!/bin/sh
for i in $( podman ps -a | grep all-in-one | awk '{print $1}' )
do
 podman rm -f $i
done

