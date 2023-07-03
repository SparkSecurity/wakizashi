# build worker binary
FROM golang:1.20-alpine as builder
RUN apk update && apk add --no-cache git
WORKDIR /src/
COPY . .
RUN go get -d -v
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o worker .

# final image
FROM debian:bullseye-slim

RUN apt update && apt install -y gnupg ca-certificates curl && \
    curl https://pkg.cloudflareclient.com/pubkey.gpg | gpg --yes --dearmor --output /usr/share/keyrings/cloudflare-warp-archive-keyring.gpg && \
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/cloudflare-warp-archive-keyring.gpg] https://pkg.cloudflareclient.com/ bullseye main" | tee /etc/apt/sources.list.d/cloudflare-client.list && \
    apt update && apt install -y cloudflare-warp && apt-get install -y wget
RUN wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add - \
    && echo "deb http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list \
    && apt-get update && apt-get -y install google-chrome-stable

COPY --from=builder /src/worker /app/worker
COPY run.sh /app/run.sh

ENV PROXY socks5://127.0.0.1:7777

ENTRYPOINT "/app/run.sh"