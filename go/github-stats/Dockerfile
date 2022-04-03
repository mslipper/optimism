FROM golang:1.17.3-alpine3.13 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git jq bash

COPY go.mod go.sum /go/github-stats/
WORKDIR /go/github-stats
RUN go mod graph | awk '{if ($1 !~ "@") print $2}' | xargs -n 1 go get
COPY ./go/github-stats/ ./
RUN make github-stats

FROM alpine:3.13

RUN apk add --no-cache ca-certificates jq curl
COPY --from=builder /go/github-stats/github-stats /usr/local/bin/

CMD ["github-stats"]
