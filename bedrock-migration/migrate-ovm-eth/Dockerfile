FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

COPY Makefile /app/Makefile
COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
COPY ./cmd /app/cmd

WORKDIR /app
RUN make migrate

FROM alpine:3.15

COPY --from=builder /app/builder /usr/local/bin

CMD migrate