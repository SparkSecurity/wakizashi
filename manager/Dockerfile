# build manager binary
FROM golang:1.20-bullseye as builder
RUN apt update && apt install -y git ca-certificates upx
WORKDIR /src/
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o manager .
RUN upx --lzma manager
RUN groupadd user && useradd -g user user

# final image
FROM busybox

COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/passwd /etc/passwd
USER user

COPY --from=builder --chown=user:user /src/manager /app/manager
COPY --from=builder --chown=user:user /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

CMD "/app/manager"
ENTRYPOINT "/app/manager"