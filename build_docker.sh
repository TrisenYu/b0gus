#!/usr/bin/env sh

# TODO: check whether docker exists in current system
# otherwise show a prompt to suggest user to install docker by themself
docker build -t b0gus_in_docker -f docker/dockerfile .
docker images
docker container ls -a
docker ps -a

# TODO: k8s
# docker run b0gus_in_docker -p22:22 -p123:123 -p20:20 -p21:21