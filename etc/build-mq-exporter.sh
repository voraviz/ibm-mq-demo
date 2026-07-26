#!/bin/sh
# Build the mq-metric-samples mq_prometheus exporter image (per-queue metrics
# like ibmmq_queue_depth). IBM publishes no prebuilt image, so we build from
# source. Target is linux/amd64 to match the MQ containers.
#
# Apple Silicon note: the upstream Dockerfile pins GOTOOLCHAIN=go1.22.8+auto on
# a go-toolset base, so `go` DOWNLOADS a fresh amd64 toolchain mid-build — and
# that download SIGSEGVs under emulation. A *prebaked* amd64 Go runs fine, so we
# base the builder on golang:1.25 (already has Go >= go.mod's 1.25) and force
# GOTOOLCHAIN=local to skip the download entirely. Works on real amd64 too.

CONTAINER_NAME=quay.io/voravitl/mq-prometheus
PLATFORM=${PLATFORM:-linux/amd64}
BASE_IMAGE=${BASE_IMAGE:-docker.io/library/golang:1.25}
# golang:1.25 is bookworm (glibc 2.36); its binary needs GLIBC_2.34, so the
# runtime must be ubi9 (glibc 2.34), not the upstream default ubi8 (glibc 2.28).
RUNTIME_IMAGE=${RUNTIME_IMAGE:-registry.access.redhat.com/ubi9-minimal:latest}
TAG=latest
IMAGE=$CONTAINER_NAME:$TAG
SRC=${SRC:-./mq-metric-samples}

CONTAINER_RUNTIME=podman
podman --version 1>/dev/null 2>&1
if [ $? -ne 0 ]; then
  CONTAINER_RUNTIME=docker
fi

echo "Cloning ibm-messaging/mq-metric-samples into $SRC ..."
rm -rf "$SRC"
git clone --depth 1 https://github.com/ibm-messaging/mq-metric-samples.git "$SRC" || exit 1

# Skip the crashing toolchain download: use the base image's baked-in Go.
sed -i.bak 's#^ENV GOTOOLCHAIN=.*#ENV GOTOOLCHAIN=local#' "$SRC/Dockerfile" && rm -f "$SRC/Dockerfile.bak"

echo "Build $IMAGE for $PLATFORM (base $BASE_IMAGE)"
$CONTAINER_RUNTIME build --platform "$PLATFORM" \
  --build-arg EXPORTER=mq_prometheus \
  --build-arg BASE_IMAGE="$BASE_IMAGE" \
  --build-arg RUNTIME_IMAGE="$RUNTIME_IMAGE" \
  -t "$IMAGE" "$SRC"
