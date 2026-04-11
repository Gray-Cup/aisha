# 🌊 Aisha

**Expose your localhost projects to every device on your network — via clean `.local` domains.**

---

## What it does

Aisha runs as a **background daemon** on your Mac and:

- Acts as a **reverse proxy** — maps `myapp.local` → `localhost:3000`
- Makes your projects reachable from any phone, tablet, or laptop on the same Wi-Fi
- Includes a **live dashboard** at `http://<your-mac-ip>:9090`
- Auto-restarts if it crashes, starts on boot via macOS launchd
- Zero external dependencies — single static binary

---

## Quick Start

### 1. Install (on your Mac)

```bash
sudo bash install.sh
```

That's it. The daemon is now running.

### 2. Edit your projects

```bash
sudo nano /usr/local/etc/aisha/config.json
```

```json
{
  "proxy_port": 80,
  "admin_port": 9090,
  "log_file": "/usr/local/var/log/aisha.log",
  "projects": [
    { "name": "myapp",     "port": 3000 },
    { "name": "api",       "port": 8080 },
    { "name": "dashboard", "port": 4000 }
  ]
}
```

After editing, restart Aisha:

```bash
sudo launchctl kickstart -k system/com.aisha.proxy
```

### 3. Access from other devices

**On your Mac** — works automatically via mDNS `.local` resolution.

**On other devices (iPhone, Windows PC, Android, etc.)** — you need to add entries to their `/etc/hosts` (or equivalent):

```
192.168.1.42    myapp.local
192.168.1.42    api.local
192.168.1.42    dashboard.local
```

Or use the helper script **from that device**:

```bash
sudo bash add_hosts.sh 192.168.1.42 myapp api dashboard
```

> 💡 Find your Mac's IP in: **System Settings → Wi-Fi → Details** or run `ipconfig getifaddr en0`

---

## Dashboard

Open in any browser on your network:

```
http://<your-mac-ip>:9090
```

Shows live status (UP/DOWN), latency, and clickable links for each project.

---

## Port 80 vs non-privileged

If you don't want to run as root, change `proxy_port` to something like `8888`:
- Update `config.json` → `"proxy_port": 8888`
- Update the plist `UserName` to your macOS username
- Access via `http://myapp.local:8888`

---

## Daemon management

| Action | Command |
|---|---|
| Status | `sudo launchctl list \| grep aisha` |
| Stop | `sudo launchctl bootout system/com.aisha.proxy` |
| Start | `sudo launchctl bootstrap system /Library/LaunchDaemons/com.aisha.proxy.plist` |
| Restart | `sudo launchctl kickstart -k system/com.aisha.proxy` |
| Logs | `tail -f /usr/local/var/log/aisha.log` |
| Uninstall | `sudo launchctl bootout system/com.aisha.proxy && sudo rm /usr/local/bin/aisha /Library/LaunchDaemons/com.aisha.proxy.plist` |

---

## Files

| File | Purpose |
|---|---|
| `aisha_mac_intel` | Binary for Intel Macs |
| `aisha_mac_apple_silicon` | Binary for M1/M2/M3/M4 Macs |
| `config.json` | Your projects & ports |
| `com.aisha.proxy.plist` | macOS daemon definition |
| `install.sh` | One-command installer |
| `add_hosts.sh` | Helper for other devices |
| `main.go` | Full source code |
