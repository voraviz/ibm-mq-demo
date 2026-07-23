#!/bin/sh
# Build static assets on the host first so esbuild runs natively (avoids the
# Go runtime 52-bit VA crash when building in-container under QEMU emulation).
echo "Installing dependencies and building static assets on host..."
npm ci && npm run build

CONTAINER_NAME=quay.io/voravitl/simple-mq-ui
PLATFORM=linux/amd64
TAG=latest
IMAGE=$CONTAINER_NAME:$TAG

echo "Build UI $IMAGE for $PLATFORM"

CONTAINER_RUNTIME=podman
podman --version 1>/dev/null 2>&1
if [ $? -ne 0 ];
then
  CONTAINER_RUNTIME=docker
fi
$CONTAINER_RUNTIME build --platform $PLATFORM -t $IMAGE -f Dockerfile .
printf "${CONTAINER_NAME}:${TAG} architectures:\n\r $ARCH"
