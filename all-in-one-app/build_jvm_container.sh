#!/bin/sh
CONTAINER_NAME=quay.io/voravitl/simple-mq-app
PLATFORM=linux/amd64 #,linux/arm64
TAG=${1:-build9} #otel,latest,hi — pass as first arg to override
DOCKERFILE=jvm-runtime #jvm,jvm-runtime,hummingbird
IMAGE=$CONTAINER_NAME:$TAG
CONTAINER_RUNTIME=podman
podman --version 1>/dev/null 2>&1
if [ $? -ne 0 ];
then
   CONTAINER_RUNTIME=docker 
fi
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
$CONTAINER_RUNTIME build --platform $PLATFORM  \
-f src/main/docker/Dockerfile.$DOCKERFILE -t $IMAGE .

