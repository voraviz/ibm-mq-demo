#!/bin/sh
CONTAINER_NAME=quay.io/voravitl/simple-mq-api-go
PLATFORM=linux/amd64,linux/arm64
TAG=latest
IMAGE=$CONTAINER_NAME:$TAG

podman manifest exists $IMAGE
if [ $? -eq 0 ];
then
  podman manifest rm $IMAGE
fi
podman manifest create $IMAGE

echo "Build Go multi-arch image $IMAGE"

CONTAINER_RUNTIME=podman
podman --version 1>/dev/null 2>&1
if [ $? -ne 0 ];
then
  CONTAINER_RUNTIME=docker
fi

$CONTAINER_RUNTIME build --platform $PLATFORM --manifest \
  $IMAGE -f Dockerfile .

$CONTAINER_RUNTIME manifest push $IMAGE

ARCH=$($CONTAINER_RUNTIME manifest inspect ${CONTAINER_NAME}:${TAG} | jq -r '.manifests[].platform.architecture')
printf "${CONTAINER_NAME}:${TAG} architectures:\n\r $ARCH"
