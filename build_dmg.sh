#!/usr/bin/env bash
# ─────────────────────────────────────────────
#  Aisha – build + package as a macOS .dmg
#
#  Usage:
#    bash build_dmg.sh [version]          # e.g. bash build_dmg.sh 1.2.0
#
#  Optional env vars:
#    SIGN_ID  – Developer ID Installer identity for notarisation-ready signing
#               e.g. SIGN_ID="Developer ID Installer: Jane Doe (XXXXXXXXXX)"
#
#  Requires: Go toolchain, Xcode Command Line Tools
#    xcode-select --install   (provides lipo, pkgbuild, productbuild, hdiutil)
# ─────────────────────────────────────────────
set -euo pipefail

# ── Config ───────────────────────────────────
VERSION="${1:-1.0.0}"
APP="aisha"
BUNDLE_ID="com.aisha.proxy"
DMG_TITLE="Aisha"
SIGN_ID="${SIGN_ID:-}"

DIST="dist"
PKG_ROOT="$DIST/pkg_root"
SCRIPTS_DIR="$DIST/pkg_scripts"
COMPONENT_PKG="$DIST/${APP}-component.pkg"
INSTALLER_PKG="$DIST/${APP}-${VERSION}.pkg"
DMG_STAGE="$DIST/dmg_stage"
OUTPUT_DMG="$DIST/${APP}-${VERSION}.dmg"
TOTAL=5

# ── Colours ──────────────────────────────────
BOLD="\033[1m"; RESET="\033[0m"; GREEN="\033[32m"; BLUE="\033[34m"; DIM="\033[2m"; RED="\033[31m"
step() { printf "\n${BOLD}${BLUE}[%d/%d]${RESET} %s\n" "$1" "$TOTAL" "$2"; }
ok()   { printf "  ${GREEN}✓${RESET} %s\n" "$1"; }
die()  { printf "\n${RED}error:${RESET} %s\n" "$1" >&2; exit 1; }

# ── Pre-flight checks ────────────────────────
printf "${BOLD}Aisha ${VERSION} — macOS installer build${RESET}\n"
printf "${DIM}%s${RESET}\n" "──────────────────────────────────────────"

for tool in go lipo pkgbuild productbuild hdiutil; do
  command -v "$tool" &>/dev/null || die "'$tool' not found. Run: xcode-select --install"
done
ok "All required tools present"

if [ ! -f "go.mod" ]; then
  die "Run this script from the Aisha repo root (go.mod not found)"
fi

# ── Clean ────────────────────────────────────
rm -rf "$DIST"
mkdir -p "$DIST"

# ─────────────────────────────────────────────
#  1. Compile universal binary
# ─────────────────────────────────────────────
step 1 "Compiling Go binaries (CGo enabled for native window)"

# webview_go requires CGo + WebKit framework.
# Cross-compiling CGo for a different arch needs an explicit target triple in CC.
SDK_PATH=$(xcrun --sdk macosx --show-sdk-path 2>/dev/null || true)

printf "  ${DIM}→ darwin/arm64${RESET}\n"
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
  go build -ldflags="-s -w" -o "$DIST/${APP}_arm64" .

printf "  ${DIM}→ darwin/amd64${RESET}\n"
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
  CC="$(xcrun --sdk macosx --find clang) -target x86_64-apple-macos10.14 -isysroot ${SDK_PATH}" \
  CGO_CFLAGS="-mmacosx-version-min=10.14" \
  CGO_LDFLAGS="-mmacosx-version-min=10.14" \
  go build -ldflags="-s -w" -o "$DIST/${APP}_intel" .

printf "  ${DIM}→ lipo → universal binary${RESET}\n"
lipo -create \
  -output "$DIST/$APP" \
  "$DIST/${APP}_intel" \
  "$DIST/${APP}_arm64"
rm "$DIST/${APP}_intel" "$DIST/${APP}_arm64"

ok "Universal binary — $(du -sh "$DIST/$APP" | cut -f1) — runs on Intel + Apple Silicon"

# ─────────────────────────────────────────────
#  2. Assemble installer payload
#     (mirrors the target filesystem layout)
# ─────────────────────────────────────────────
step 2 "Assembling installer payload"

mkdir -p "$PKG_ROOT/usr/local/bin"
mkdir -p "$PKG_ROOT/Library/LaunchDaemons"

install -m 755 "$DIST/$APP"           "$PKG_ROOT/usr/local/bin/aisha"
install -m 644 com.aisha.proxy.plist "$PKG_ROOT/Library/LaunchDaemons/com.aisha.proxy.plist"

ok "Payload: /usr/local/bin/aisha  +  /Library/LaunchDaemons/com.aisha.proxy.plist"

# ── Aisha.app bundle → /Applications ───────
APP_BUNDLE="$PKG_ROOT/Applications/Aisha.app/Contents"
mkdir -p "$APP_BUNDLE/MacOS"
mkdir -p "$APP_BUNDLE/Resources"

# The binary IS the app bundle executable — no shell wrapper, so macOS
# shows a single icon and associates the window with the .app correctly.
cp "$DIST/$APP" "$APP_BUNDLE/MacOS/Aisha"
chmod +x "$APP_BUNDLE/MacOS/Aisha"

cat > "$APP_BUNDLE/Info.plist" << INFOPLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>         <string>Aisha</string>
  <key>CFBundleIdentifier</key>         <string>com.aisha.app</string>
  <key>CFBundleName</key>               <string>Aisha</string>
  <key>CFBundleDisplayName</key>        <string>Aisha</string>
  <key>CFBundleVersion</key>            <string>${VERSION}</string>
  <key>CFBundleShortVersionString</key> <string>${VERSION}</string>
  <key>CFBundlePackageType</key>        <string>APPL</string>
  <key>CFBundleSignature</key>          <string>????</string>
  <key>LSMinimumSystemVersion</key>     <string>10.14</string>
  <key>NSHighResolutionCapable</key>    <true/>
  <key>NSHumanReadableCopyright</key>   <string>Aisha ${VERSION}</string>
  <!-- Allow WKWebView to load local HTTP content -->
  <key>NSAppTransportSecurity</key>
  <dict>
    <key>NSAllowsLocalNetworking</key> <true/>
  </dict>
</dict>
</plist>
INFOPLIST

ok "App bundle: /Applications/Aisha.app  (native WKWebView window)"

# ─────────────────────────────────────────────
#  3. Pre/postinstall scripts
# ─────────────────────────────────────────────
step 3 "Writing installer scripts"

mkdir -p "$SCRIPTS_DIR"

# preinstall – gracefully stop any running daemon before overwriting the binary
cat > "$SCRIPTS_DIR/preinstall" << 'PREINSTALL'
#!/bin/bash
launchctl bootout system/com.aisha.proxy 2>/dev/null || true
exit 0
PREINSTALL

# postinstall – fix permissions, seed config, start daemon
cat > "$SCRIPTS_DIR/postinstall" << 'POSTINSTALL'
#!/bin/bash
set -e

# ── Permissions ──────────────────────────────
chmod 755 /usr/local/bin/aisha
chown root:wheel /Library/LaunchDaemons/com.aisha.proxy.plist
chmod 644 /Library/LaunchDaemons/com.aisha.proxy.plist
chmod -R 755 /Applications/Aisha.app
chmod +x /Applications/Aisha.app/Contents/MacOS/Aisha

# ── Log files ────────────────────────────────
mkdir -p /usr/local/var/log
touch /usr/local/var/log/aisha.log
touch /usr/local/var/log/aisha-error.log

# ── Default config (only if not already present) ─
if [ ! -f /usr/local/etc/aisha/config.json ]; then
  mkdir -p /usr/local/etc/aisha
  cat > /usr/local/etc/aisha/config.json << 'CONFIG'
{
  "proxy_port": 80,
  "admin_port": 9090,
  "log_file": "/usr/local/var/log/aisha.log",
  "projects": [
    { "name": "myapp", "port": 3000 },
    { "name": "api",   "port": 8080 }
  ]
}
CONFIG
fi

# ── Start daemon ─────────────────────────────
launchctl bootstrap system /Library/LaunchDaemons/com.aisha.proxy.plist

IP=$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || echo "your-mac-ip")
echo ""
echo "Aisha is running!"
echo ""
echo "  Dashboard → http://${IP}:9090"
echo "  Config    → sudo nano /usr/local/etc/aisha/config.json"
echo "  Logs      → tail -f /usr/local/var/log/aisha.log"
echo ""
exit 0
POSTINSTALL

chmod +x "$SCRIPTS_DIR/preinstall" "$SCRIPTS_DIR/postinstall"
ok "preinstall + postinstall scripts written"

# ─────────────────────────────────────────────
#  4. Build .pkg installer
# ─────────────────────────────────────────────
step 4 "Building installer package"

# Component package (raw payload + scripts)
pkgbuild \
  --root       "$PKG_ROOT" \
  --scripts    "$SCRIPTS_DIR" \
  --identifier "$BUNDLE_ID" \
  --version    "$VERSION" \
  --ownership  recommended \
  "$COMPONENT_PKG"

# Distribution package (final, optionally signed)
if [ -n "$SIGN_ID" ]; then
  printf "  ${DIM}→ signing with: %s${RESET}\n" "$SIGN_ID"
  productbuild \
    --package    "$COMPONENT_PKG" \
    --identifier "$BUNDLE_ID" \
    --version    "$VERSION" \
    --sign       "$SIGN_ID" \
    "$INSTALLER_PKG"
else
  productbuild \
    --package    "$COMPONENT_PKG" \
    --identifier "$BUNDLE_ID" \
    --version    "$VERSION" \
    "$INSTALLER_PKG"
fi

rm "$COMPONENT_PKG"
ok "Package ready — $(du -sh "$INSTALLER_PKG" | cut -f1)"

# ─────────────────────────────────────────────
#  5. Create .dmg
# ─────────────────────────────────────────────
step 5 "Creating DMG"

mkdir -p "$DMG_STAGE"
cp "$INSTALLER_PKG" "$DMG_STAGE/"
cp add_hosts.sh     "$DMG_STAGE/"

hdiutil create \
  -volname   "$DMG_TITLE" \
  -srcfolder "$DMG_STAGE" \
  -ov \
  -format    UDZO \
  -imagekey  zlib-level=9 \
  "$OUTPUT_DMG" \
  -quiet

# ── Cleanup intermediates ────────────────────
rm -rf "$PKG_ROOT" "$SCRIPTS_DIR" "$DMG_STAGE" "$DIST/$APP" "$INSTALLER_PKG"

# ── Summary ──────────────────────────────────
printf "\n${BOLD}${GREEN}Done!${RESET}\n\n"
printf "  %-10s %s\n" "Output:" "$OUTPUT_DMG"
printf "  %-10s %s\n" "Size:"   "$(du -sh "$OUTPUT_DMG" | cut -f1)"
printf "\n"
printf "  Mount the DMG and double-click ${BOLD}Aisha-${VERSION}.pkg${RESET}\n"
printf "  The installer will ask for your admin password\n"
printf "  (needs root to bind port 80 and register the launchd daemon).\n\n"

if [ -z "$SIGN_ID" ]; then
  printf "  ${DIM}Tip: to notarise and avoid Gatekeeper warnings, set:${RESET}\n"
  printf "  ${DIM}SIGN_ID=\"Developer ID Installer: You (XXXXXXXXXX)\" bash build_dmg.sh${RESET}\n\n"
fi
