# This Dockerfile builds all the dependencies needed by the monorepo, and should
# be used to build any of the follow-on services
#
# ### BASE: Install deps
# We do not use Alpine because there's a regression causing it to be very slow
# when used with typescript/hardhat: https://github.com/nomiclabs/hardhat/issues/1219
FROM node:14-alpine

RUN apk add --no-cache git curl python3 bash jq

# Pre-download the compilers so that they do not need to be downloaded inside
# the image when building
ARG VERSION=v0.7.6
ARG SOLC_VERSION=${VERSION}+commit.7338295f
ARG SOLC_UPSTREAM=https://github.com/ethereum/solc-bin/raw/gh-pages/linux-amd64/solc-linux-amd64-${SOLC_VERSION}

ADD $SOLC_UPSTREAM ./solc

WORKDIR /optimism
COPY *.json yarn.lock ./
COPY packages/core-utils/package.json ./packages/core-utils/package.json
COPY packages/common-ts/package.json ./packages/common-ts/package.json
COPY packages/hardhat-ovm/package.json ./packages/hardhat-ovm/package.json
COPY packages/contracts/package.json ./packages/contracts/package.json
COPY packages/data-transport-layer/package.json ./packages/data-transport-layer/package.json
COPY packages/batch-submitter/package.json ./packages/batch-submitter/package.json
COPY packages/message-relayer/package.json ./packages/message-relayer/package.json
COPY packages/replica-healthcheck/package.json ./packages/replica-healthcheck/package.json
COPY integration-tests/package.json ./integration-tests/package.json
RUN yarn install --frozen-lockfile

COPY packages ./packages
COPY integration-tests ./integration-tests

RUN yarn build
