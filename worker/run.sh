#!/bin/bash
/bin/warp-svc &
sleep 5
/bin/warp-cli --accept-tos register
/bin/warp-cli --accept-tos set-mode proxy
/bin/warp-cli --accept-tos set-proxy-port 7777
/bin/warp-cli --accept-tos connect
/app/worker
