#!/bin/sh
CONTAINER_NAME=quay.io/voravitl/simple-mq-app
PLATFORM=linux/amd64 #,linux/arm64
TAG=build9 #otel,latest,hi
DOCKERFILE=jvm-runtime #jvm,jvm-runtime,hummingbird
IMAGE=$CONTAINER_NAME:$TAG
CONTAINER_RUNTIME=podman
podman --version 1>/dev/null 2>&1
if [ $? -ne 0 ];
then
   CONTAINER_RUNTIME=docker 
fi
mvn clean package -DskipTests=true
echo "Build with Dockerfile.$DOCKERFILE tag $TAG"
$CONTAINER_RUNTIME build --platform $PLATFORM  \
-f src/main/docker/Dockerfile.$DOCKERFILE -t $IMAGE .

