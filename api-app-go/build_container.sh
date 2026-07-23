#!/bin/sh
CONTAINER_NAME=quay.io/voravitl/simple-mq-api-go
# amd64-only: IBM ships the MQ redistributable C client for LinuxX64 only,
# and mq-golang requires that native client (CGO). No LinuxARM64 redist exists.
PLATFORM=linux/amd64
TAG=latest
IMAGE=$CONTAINER_NAME:$TAG

echo "Build Golang $IMAGE"

CONTAINER_RUNTIME=podman
podman --version 1>/dev/null 2>&1
if [ $? -ne 0 ];
then
  CONTAINER_RUNTIME=docker
fi
$CONTAINER_RUNTIME build --platform $PLATFORM -t $IMAGE -f Dockerfile . 
printf "${CONTAINER_NAME}:${TAG} architectures:\n\r $ARCH"
