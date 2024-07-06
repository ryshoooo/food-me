#!/usr/bin/env bash

set -ex

function cleanup() {
    set +e
    docker stop $(docker ps -a -q)
    docker system prune -f
    docker volume prune -f
    set -e
}

function wait() {
    SECONDS=0
    until curl --silent --output /dev/null localhost:8080; do
      sleep 5;
      if [ ${SECONDS} -gt 180 ]; then
        echo "Timeout exceeded";
        exit 1;
      fi
    done
}

trap cleanup err exit

# Get the script location
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Go to the examples folder
cd $DIR/../examples

# Run each example
for d in */ ; do
    cd $d
    echo "Running example $d"
    docker build -f Dockerfile -t foodme:e2e .
    docker compose up -d --build
    wait
    docker run --network host foodme:e2e
    docker compose down
    cd ..
done