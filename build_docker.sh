#!/usr/bin/env sh

docker build -t b0gus_in_docker -f docker/dockerfile .
docker images
docker container ls -a
docker ps -a