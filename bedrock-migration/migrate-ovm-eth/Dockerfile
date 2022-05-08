FROM golang:1.18.0-alpine3.15 as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

COPY ./cmd /app/cmd
COPY Makefile /app/Makefile

RUN make migrate

FROM alpine:3.15

COPY --from=builder /app/builder /usr/local/bin

CMD migrate