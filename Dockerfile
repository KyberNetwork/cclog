## BUILDER
FROM golang:1.22 as builder

WORKDIR /src

COPY . .

RUN go build -o app ./cmd/cclog-server

FROM debian:bullseye

RUN apt-get update && \
    apt install -y ca-certificates && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /cmd

COPY --from=builder /src/app /app/cmd

ENTRYPOINT /app/cmd
