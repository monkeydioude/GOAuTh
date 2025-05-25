# BUILD PART
FROM golang:1.24.0-alpine AS builder

RUN apk add --no-cache bash

WORKDIR /app
COPY . .
RUN env

RUN go build -C bin/GOAuTh -o /GOAuTh

# RUN PART
FROM alpine:latest

RUN apk add --no-cache bash postgresql-client

WORKDIR /app

COPY --from=builder /app/scripts/docker/entrypoint.sh .
COPY --from=builder /GOAuTh .

ENTRYPOINT [ "./entrypoint.sh" ]
CMD ["./GOAuTh"]
