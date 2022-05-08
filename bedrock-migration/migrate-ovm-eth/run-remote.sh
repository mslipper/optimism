#!/usr/bin/env bash

ssh archi-local bas <<EOF
cd /home/ubuntu/dev/optimism
git fetch
git checkout migrator
git pull
cd bedrock-migration/migrate-ovm-eth
docker build -t mslipper/migrator:latest .
EOF