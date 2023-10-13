#!/bin/bash
if [ "$PROXY_MODE" = "warp" ]; then
    /bin/warp-svc &
    sleep 5
    /bin/warp-cli --accept-tos register
    /bin/warp-cli --accept-tos set-mode proxy
    /bin/warp-cli --accept-tos set-proxy-port 7777
    /bin/warp-cli --accept-tos connect
    PROXY=socks5://127.0.0.1:7777 /app/worker
elif [ "$PROXY_MODE" = "vmess" ]; then
    sed -i "s/vmess_server_ip/$VMESS_SERVER_IP/g" /app/clash/config.yaml
    sed -i "s/vmess_uuid/$VMESS_UUID/g" /app/clash/config.yaml
    pm2 start clash -d /app/clash
    PROXY=socks5://127.0.0.1:7890 /app/worker
else
    /app/worker
fi