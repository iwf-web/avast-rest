#!/bin/bash
set -e

if [ -n "$AVAST_ACTIVATION_CODE" ]; then
    echo "Activating Avast license..."
    avastlic -o /etc/avast/license.avastlic -c "$AVAST_ACTIVATION_CODE"
    echo "License generated - please save the generated file and mount it next time into /etc/avast/license.avastlic when you start the container"
    exit 0
elif [ ! -f /etc/avast/license.avastlic ]; then
    echo "ERROR: No license available."
    echo "       Set AVAST_ACTIVATION_CODE env var or mount a license file:"
    echo "       -v /path/to/license.avastlic:/etc/avast/license.avastlic"
    exit 1
fi

echo "Downloading Avast virus definitions..."
/usr/lib/avast/vpsupdate

echo "Starting Avast daemon..."
/usr/bin/avast &

for i in $(seq 1 30); do
    [ -S /var/run/avast/scan.sock ] && break
    sleep 1
done
[ -S /var/run/avast/scan.sock ] || { echo "ERROR: Avast daemon socket not ready after 30s"; exit 1; }

(
    while true; do
        sleep 14400
        echo "Updating Avast virus definitions..."
        /usr/lib/avast/vpsupdate && \
            kill -HUP "$(cat /var/run/avast/avast.pid)" 2>/dev/null || true
    done
) &

echo "Starting antivirus-rest..."
exec /usr/bin/av-rest
