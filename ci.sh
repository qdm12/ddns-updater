#!/bin/bash

if [ "$TRAVIS_PULL_REQUEST" = "true" ]; then
  docker buildx build --platform=linux/amd64,linux/386 --progress plain .
  return $?
fi
echo $DOCKER_PASSWORD | docker login -u qmcgaw --password-stdin &> /dev/null
TAG="$TRAVIS_BRANCH"
if [ "$TAG" = "master" ]; then
  TAG="${TRAVIS_TAG:-latest}"
fi
echo "Building Docker images for \"$DOCKER_REPO:$TAG\""
docker buildx build \
    --platform=linux/amd64,linux/386 \
    --progress plain \
    --build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
    --build-arg VCS_REF=`git rev-parse --short HEAD` \
    --build-arg VERSION=$TAG \
    -t $DOCKER_REPO:$TAG \
    --push \
    .
