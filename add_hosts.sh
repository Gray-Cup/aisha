#!/bin/bash
# ─────────────────────────────────────────────
#  Twisha – Hosts Helper
#  Adds .local entries to /etc/hosts on OTHER
#  devices so they can find your Mac's projects.
#
#  Run on EACH device that needs access:
#    sudo bash add_hosts.sh 192.168.1.42 myapp api dashboard
# ─────────────────────────────────────────────

MAC_IP="$1"
shift
NAMES=("$@")

if [ -z "$MAC_IP" ] || [ ${#NAMES[@]} -eq 0 ]; then
  echo "Usage: sudo bash add_hosts.sh <mac-ip> <project1> <project2> ..."
  echo "Example: sudo bash add_hosts.sh 192.168.1.42 myapp api dashboard"
  exit 1
fi

echo "# Twisha – added $(date)" >> /etc/hosts
for name in "${NAMES[@]}"; do
  echo "$MAC_IP    $name.local" >> /etc/hosts
  echo "Added: $MAC_IP  $name.local"
done

echo ""
echo "✅ Done. Try: ping myapp.local"
