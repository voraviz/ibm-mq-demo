#!/bin/sh
CONTAINER_NAME=quay.io/voravitl/simple-mq-app
PLATFORM=linux/amd64,linux/arm64
TAG=${1:-multi-arch} #otel,latest,hi — pass as first arg to override
DOCKERFILE=jvm-runtime #jvm,jvm-runtime,hummingbird
IMAGE=$CONTAINER_NAME:$TAG
CONTAINER_RUNTIME=podman
podman --version 1>/dev/null 2>&1
if [ $? -ne 0 ];
then
   CONTAINER_RUNTIME=docker 
fi
$CONTAINER_RUNTIME manifest exists $IMAGE 
if [ $? -eq 0 ];
then
 $CONTAINER_RUNTIME manifest rm $IMAGE
fi
$CONTAINER_RUNTIME manifest create $IMAGE 
echo "Build with Dockerfile.$DOCKERFILE tag $TAG"
MVN_PROFILE=""
if [ "$TAG" = "otel" ];
then
   MVN_PROFILE="-Dquarkus.profile=otel"
fi
mvn clean package -DskipTests=true $MVN_PROFILE
if [ $? -ne 0 ];
then
   echo "Maven build failed, aborting"
   exit 1
fi
$CONTAINER_RUNTIME build --platform $PLATFORM  --manifest \
$IMAGE -f src/main/docker/Dockerfile.$DOCKERFILE  .
$CONTAINER_RUNTIME manifest push $IMAGE
if [ $? -eq 0 ];
then
    # Use `manifest inspect` (not `inspect`): `inspect` resolves a manifest to a
    # single image — the host arch — so it always prints just arm64 here.
    # `manifest inspect` lists every architecture actually in the manifest list.
    ARCH=$($CONTAINER_RUNTIME manifest inspect ${CONTAINER_NAME}:${TAG} | jq -r '.manifests[].platform.architecture')
    printf "${CONTAINER_NAME}:${TAG} architectures:\n%s\n" "$ARCH"
fi
