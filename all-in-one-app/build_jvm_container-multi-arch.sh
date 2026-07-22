#!/bin/sh
CONTAINER_NAME=quay.io/voravitl/simple-mq-app
PLATFORM=linux/amd64,linux/arm64
TAG=latest #otel,latest
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
mvn clean package -DskipTests=true
$CONTAINER_RUNTIME build --platform $PLATFORM  --manifest \
$IMAGE -f src/main/docker/Dockerfile.$DOCKERFILE  .
$CONTAINER_RUNTIME manifest push $IMAGE
ARCH=$($CONTAINER_RUNTIME manifest inspect ${CONTAINER_NAME}:${TAG} | jq -r '.manifests[].platform.architecture')
printf "${CONTAINER_NAME}:${TAG} architectures:\n\r $ARCH"
# if [ $? -eq 0 ];
# then
#    ARCH=$($CONTAINER_RUNTIME inspect ${CONTAINER_NAME}:${TAG} | jq '.[0].Architecture' | sed 's/\"//g')
#    printf "${CONTAINER_NAME}:${TAG} architecture is $ARCH"
# fi
