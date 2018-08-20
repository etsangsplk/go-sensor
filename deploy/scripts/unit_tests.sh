#!/bin/sh

set -e

export IMAGE=$1

# Unit tests
echo "Running units"
docker-compose \
    --project-directory `pwd` \
    -f deploy/unit_tests.yml \
    up \
    --abort-on-container-exit \
    --exit-code-from units \
    --force-recreate \
    --no-build \
    --renew-anon-volumes
