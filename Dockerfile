FROM alpine:3.12.0
WORKDIR /app
COPY ./cmd/cclog-server/cclog-server /app/cmd
CMD ["/app/cmd"]