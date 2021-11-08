#!/bin/bash
CONTAINER=l2geth

RETRIES=30
i=0
until docker-compose logs l2geth | grep -q "Starting Sequencer Loop";
do
    sleep 3
    if [ $i -eq $RETRIES ]; then
        echo 'Timed out waiting for sequencer'
        echo 'Sequencer logs:'
        docker logs ops_l2geth_1 -n 500
        exit 1
    fi
    echo 'Waiting for sequencer...'
    ((i=i+1))
done
