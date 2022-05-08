#!/usr/bin/env bash

ssh archi-local bash <<EOF
cd /home/ubuntu/dev/optimism
git fetch
git checkout migrator
git pull
cd bedrock-migration/migrate-ovm-eth
docker build -t mslipper/migrator:latest .
docker run -v op-replica_geth:/geth mslipper/migrator:latest /geth dummy
EOF