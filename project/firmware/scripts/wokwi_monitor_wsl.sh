#!/usr/bin/env bash
set -euo pipefail

# Wokwi RFC2217 server runs in Windows (VS Code extension host), not inside WSL.
# In WSL, the current Windows host address is the nameserver in /etc/resolv.conf.
WIN_HOST="$(awk '/nameserver/ {print $2; exit}' /etc/resolv.conf)"

if [[ -z "${WIN_HOST}" ]]; then
  echo "Could not detect Windows host IP from /etc/resolv.conf" >&2
  exit 1
fi

echo "Using Windows host: ${WIN_HOST}"
echo "Connecting to: rfc2217://${WIN_HOST}:4000"

cd "$(dirname "$0")/.."
pio device monitor -e t-a7670-wokwi --port "rfc2217://${WIN_HOST}:4000" --filter direct --filter time
