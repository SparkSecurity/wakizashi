# build manager binary
FROM golang:1.20-alpine as builder
RUN apk update && apk add --no-cache git ca-certificates upx
WORKDIR /src/
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o manager .
RUN upx --lzma manager

# final image
FROM busybox

COPY --from=builder /src/manager /app/manager
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD "/app/manager"
ENTRYPOINT "/app/manager"