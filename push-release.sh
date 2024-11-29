#!/usr/bin/env bash

VERSION=$1
export VERSION
echo "${VERSION}" > version

REPOSITORY="kuznetcovay/go-prom-proxy"
export REPOSITORY

if [ -z "$VERSION" ]; then
    echo "VERSION is empty. Exiting with code 1."
    exit 1
fi

if [ "$2" == "push" ]; then
    echo "pushing"
    docker buildx build --push --platform=linux/amd64,linux/arm64 -t "${REPOSITORY}:${VERSION}" . || \
        { echo "docker buildx push failed. Exiting with code 1."; exit 1; }
else
    echo "loading"
    docker buildx build --load --platform=linux/amd64 --progress=plain -t "${REPOSITORY}:${VERSION}" . || \
        { echo "docker buildx load failed. Exiting with code 1."; exit 1; }
fi


git add -A .|| \
        { echo "git add failed. Exiting with code 1."; exit 1; }
git commit -m "release $VERSION" || \
        { echo "git commit failed. Exiting with code 1."; exit 1; }
git tag -a "$VERSION" -m "release $VERSION" || \
        { echo "git tag failed. Exiting with code 1."; exit 1; }
git push -u origin main --tags || \
        { echo "git push failed. Exiting with code 1."; exit 1; }
