#!/bin/bash
# ─────────────────────────────────────────────
#  Twisha – macOS install script
#  Run with: sudo bash install.sh
# ─────────────────────────────────────────────
set -e

BINARY_INTEL="twisha_mac_intel"
BINARY_APPLE="twisha_mac_apple_silicon"
INSTALL_BIN="/usr/local/bin/twisha"
INSTALL_CFG_DIR="/usr/local/etc/twisha"
INSTALL_CFG="$INSTALL_CFG_DIR/config.json"
PLIST_SRC="com.twisha.proxy.plist"
PLIST_DST="/Library/LaunchDaemons/com.twisha.proxy.plist"
LOG_DIR="/usr/local/var/log"

# Detect arch
ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ]; then
  BINARY="$BINARY_APPLE"
  echo "→ Detected Apple Silicon (arm64)"
else
  BINARY="$BINARY_INTEL"
  echo "→ Detected Intel (x86_64)"
fi

# Install binary
echo "→ Installing binary to $INSTALL_BIN"
cp "$BINARY" "$INSTALL_BIN"
chmod +x "$INSTALL_BIN"

# Install config (don't overwrite existing)
mkdir -p "$INSTALL_CFG_DIR"
if [ ! -f "$INSTALL_CFG" ]; then
  echo "→ Installing default config to $INSTALL_CFG"
  cp config.json "$INSTALL_CFG"
else
  echo "→ Config already exists at $INSTALL_CFG (skipping)"
fi

# Log dir
mkdir -p "$LOG_DIR"
touch "$LOG_DIR/twisha.log" "$LOG_DIR/twisha-error.log"

# Install plist
echo "→ Installing LaunchDaemon plist"
cp "$PLIST_SRC" "$PLIST_DST"
chown root:wheel "$PLIST_DST"
chmod 644 "$PLIST_DST"

# Stop existing if running
launchctl bootout system/com.twisha.proxy 2>/dev/null || true

# Load daemon
echo "→ Loading daemon"
launchctl bootstrap system "$PLIST_DST"

echo ""
echo "✅ Twisha installed and running as a background daemon!"
echo ""
echo "   Edit config: sudo nano $INSTALL_CFG"
echo "   View logs:   tail -f $LOG_DIR/twisha.log"
echo "   Dashboard:   http://$(ipconfig getifaddr en0):9090"
echo ""
echo "   Other devices on your network can reach your projects at:"
grep -o '"name": *"[^"]*"' "$INSTALL_CFG" | sed 's/"name": *"//;s/"//' | \
  while read name; do echo "     http://$name.local"; done
echo ""
echo "   Stop:    sudo launchctl bootout system/com.twisha.proxy"
echo "   Start:   sudo launchctl bootstrap system $PLIST_DST"
echo "   Restart: sudo launchctl kickstart -k system/com.twisha.proxy"
