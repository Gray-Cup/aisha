package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ─────────────────────────────────────────────
//  Dashboard HTML
// ─────────────────────────────────────────────

func dashboardPage(cfg Config, stat *healthStatus, ip string) string {
	projectsJSON, _ := json.Marshal(func() []map[string]interface{} {
		out := make([]map[string]interface{}, 0, len(cfg.Projects))
		for _, p := range cfg.Projects {
			out = append(out, map[string]interface{}{
				"name":    p.Name,
				"port":    p.Port,
				"command": p.Command,
				"dir":     p.Dir,
			})
		}
		return out
	}())

	stat.mu.RLock()
	defer stat.mu.RUnlock()

	var projectRows strings.Builder
	for _, p := range cfg.Projects {
		up := stat.up[p.Name]
		lat := stat.latency[p.Name]
		latStr := ""
		if up && lat > 0 {
			latStr = fmt.Sprintf(`<span class="lat">%dms</span>`, lat.Milliseconds())
		}
		statusDot := `<span class="dot dot-down" title="Down"></span>`
		if up {
			statusDot = `<span class="dot dot-up" title="Up"></span>`
		}
		cmdStr := p.Command
		if cmdStr == "" {
			cmdStr = "No command"
		}
		projectRows.WriteString(fmt.Sprintf(`
		<div class="proj-row" id="pr-%s" onclick="selectProject('%s')" data-name="%s">
		  <div class="proj-left">
		    %s
		    <div class="proj-meta">
		      <div class="proj-name">%s</div>
		      <div class="proj-cmd" id="cmd-%s">%s</div>
		    </div>
		  </div>
		  <div class="proj-right">
		    %s
		    <div class="proj-port">:%d</div>
		    <button class="btn-shield" id="shield-%s" onclick="event.stopPropagation();showMACPanel('%s')" title="MAC Allowlist">
		      <svg width="11" height="13" viewBox="0 0 12 14" fill="none"><path d="M6 0.5L0.5 2.5V7C0.5 10.05 2.9 12.9 6 13.85C9.1 12.9 11.5 10.05 11.5 7V2.5L6 0.5Z" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"/></svg>
		    </button>
		  </div>
		</div>`,
			p.Name, p.Name, p.Name,
			statusDot,
			p.Name, p.Name, cmdStr,
			latStr, p.Port,
			p.Name, p.Name))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="color-scheme" content="light">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Twisha</title>
<style>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

:root {
  --bg:          #f3f3f3;
  --bg2:         #ffffff;
  --bg3:         #f0f0f0;
  --bg4:         #e8e8e8;
  --border:      #e0e0e0;
  --border2:     #c8c8c8;
  --text:        #383838;
  --text-dim:    #717171;
  --text-bright: #1a1a1a;
  --blue:        #005fb8;
  --blue-bg:     #ddeeff;
  --green:       #107c10;
  --green-bg:    #dff6dd;
  --red:         #c5271f;
  --red-bg:      #fce8e8;
  --yellow:      #7d5c00;
  --accent:      #005fb8;
  --accent-h:    #004c99;
  --font:        -apple-system, BlinkMacSystemFont, 'SF Pro Text', 'Helvetica Neue', sans-serif;
  --mono:        'SF Mono', ui-monospace, 'Cascadia Code', 'Fira Code', monospace;
  --sidebar-w:   248px;
  --titlebar-h:  38px;
  --statusbar-h: 22px;
}

html, body { height: 100%%; overflow: hidden; }
body {
  font-family: var(--font);
  font-size: 13px;
  background: var(--bg);
  color: var(--text);
  -webkit-font-smoothing: antialiased;
  display: flex;
  flex-direction: column;
  user-select: none;
  -webkit-user-select: none;
}

/* ── Titlebar ── */
.titlebar {
  height: var(--titlebar-h);
  background: var(--bg2);
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 12px 0 80px;
  gap: 8px;
  flex-shrink: 0;
  -webkit-app-region: drag;
}
.titlebar-logo {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--accent);
}
.titlebar-sep { width: 1px; height: 14px; background: var(--border2); flex-shrink: 0; }
.titlebar-meta { font-size: 11px; color: var(--text-dim); }
.titlebar-actions {
  margin-left: auto;
  display: flex;
  gap: 4px;
  -webkit-app-region: no-drag;
}
.tb-btn {
  font-family: var(--font);
  font-size: 11px;
  font-weight: 500;
  padding: 3px 10px;
  border-radius: 4px;
  border: 1px solid var(--border2);
  background: transparent;
  color: var(--text-dim);
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
  -webkit-app-region: no-drag;
}
.tb-btn:hover { background: var(--bg4); color: var(--text); }

/* ── Layout ── */
.layout {
  display: grid;
  grid-template-columns: var(--sidebar-w) 1fr;
  flex: 1;
  overflow: hidden;
}

/* ── Sidebar ── */
.sidebar {
  background: var(--bg2);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}
.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 12px 6px;
  flex-shrink: 0;
}
.sidebar-section-label {
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-dim);
}
.btn-new-proj {
  width: 20px;
  height: 20px;
  border-radius: 4px;
  border: 1px solid var(--border2);
  background: transparent;
  color: var(--text-dim);
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  transition: background 0.1s, color 0.1s, border-color 0.1s;
}
.btn-new-proj:hover { background: var(--blue-bg); color: var(--accent); border-color: var(--accent); }

.proj-list {
  flex: 1;
  overflow-y: auto;
  padding: 2px 0;
}
.proj-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  cursor: pointer;
  border-radius: 4px;
  margin: 1px 4px;
  gap: 6px;
  transition: background 0.1s;
}
.proj-row:hover { background: var(--bg3); }
.proj-row.selected { background: var(--blue-bg); }
.proj-left { display: flex; align-items: center; gap: 7px; min-width: 0; flex: 1; }
.proj-meta { min-width: 0; }
.proj-name { font-size: 12px; font-weight: 600; color: var(--text-bright); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.proj-cmd { font-size: 10px; color: var(--text-dim); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-family: var(--mono); margin-top: 1px; }
.proj-right { display: flex; align-items: center; gap: 5px; flex-shrink: 0; }
.proj-port { font-size: 10px; color: var(--text-dim); font-family: var(--mono); }

/* Status dots */
.dot { display: inline-block; width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.dot-up      { background: #107c10; box-shadow: 0 0 4px rgba(16,124,16,.35); }
.dot-down    { background: var(--border2); }
.dot-managed { background: var(--accent); box-shadow: 0 0 4px rgba(0,95,184,.35); }

.lat { font-size: 10px; color: var(--text-dim); font-variant-numeric: tabular-nums; font-family: var(--mono); }

/* Shield button */
.btn-shield {
  background: none;
  border: 1px solid var(--border);
  border-radius: 3px;
  padding: 2px 4px;
  cursor: pointer;
  color: var(--text-dim);
  display: flex;
  align-items: center;
  transition: color 0.12s, border-color 0.12s, background 0.12s;
}
.btn-shield:hover { border-color: var(--accent); color: var(--accent); }
.btn-shield.shield-on { border-color: var(--accent); color: var(--accent); background: var(--blue-bg); }

/* Sidebar footer */
.sidebar-footer {
  border-top: 1px solid var(--border);
  padding: 8px;
  flex-shrink: 0;
}
.net-info { display: flex; flex-direction: column; gap: 3px; }
.net-row { display: flex; justify-content: space-between; font-size: 10px; }
.net-label { color: var(--text-dim); }
.net-val { color: var(--text); font-family: var(--mono); }

/* ── Main panel ── */
.main-panel {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--bg);
}

/* Tab bar */
.tab-bar {
  display: flex;
  border-bottom: 1px solid var(--border);
  background: var(--bg2);
  flex-shrink: 0;
}
.tab {
  font-family: var(--font);
  font-size: 12px;
  font-weight: 500;
  padding: 8px 16px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--text-dim);
  cursor: pointer;
  transition: color 0.12s;
  display: flex;
  align-items: center;
  gap: 5px;
}
.tab:hover { color: var(--text); }
.tab.active { color: var(--text-bright); border-bottom-color: var(--accent); }
.tab-badge {
  font-size: 10px;
  font-weight: 700;
  background: var(--accent);
  color: #fff;
  border-radius: 999px;
  min-width: 15px;
  height: 15px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 3px;
}

.tab-content { display: none; flex: 1; overflow: hidden; flex-direction: column; }
.tab-content.active { display: flex; }

/* ── Project detail panel ── */
.detail-pane {
  flex: 1;
  overflow-y: auto;
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}
.detail-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-bright);
  letter-spacing: -0.02em;
}
.detail-subtitle { font-size: 11px; color: var(--text-dim); margin-top: 2px; }

/* Card */
.card {
  background: var(--bg2);
  border: 1px solid var(--border);
  border-radius: 6px;
  overflow: hidden;
}
.card-header {
  padding: 10px 14px;
  border-bottom: 1px solid var(--border);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--text-dim);
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--bg3);
}
.card-body { padding: 14px; }

/* Stats row */
.stats-row { display: flex; gap: 1px; border-radius: 6px; overflow: hidden; border: 1px solid var(--border); }
.stat-cell {
  flex: 1;
  background: var(--bg2);
  padding: 10px 14px;
  border-right: 1px solid var(--border);
}
.stat-cell:last-child { border-right: none; }
.stat-value { font-size: 22px; font-weight: 700; color: var(--text-bright); font-variant-numeric: tabular-nums; letter-spacing: -0.03em; }
.stat-label { font-size: 10px; color: var(--text-dim); margin-top: 2px; text-transform: uppercase; letter-spacing: 0.07em; }

/* Process controls */
.proc-controls { display: flex; align-items: center; gap: 8px; }
.proc-status { display: flex; align-items: center; gap: 5px; font-size: 11px; }
.proc-status-label { color: var(--text-dim); }
.proc-status-val { font-weight: 600; }
.proc-status-val.running { color: var(--accent); }
.proc-status-val.stopped { color: var(--text-dim); }

/* Buttons */
.btn {
  font-family: var(--font);
  font-size: 11px;
  font-weight: 600;
  padding: 5px 12px;
  border-radius: 4px;
  border: none;
  cursor: pointer;
  transition: background 0.12s, opacity 0.12s;
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.btn-primary { background: var(--accent); color: #fff; }
.btn-primary:hover { background: var(--accent-h); }
.btn-primary:disabled { opacity: 0.45; cursor: not-allowed; }
.btn-danger { background: var(--red-bg); color: var(--red); border: 1px solid #e8b4b4; }
.btn-danger:hover { background: #f8d5d5; }
.btn-ghost { background: var(--bg3); color: var(--text); border: 1px solid var(--border); }
.btn-ghost:hover { background: var(--bg4); }
.btn-success { background: var(--green-bg); color: var(--green); border: 1px solid #afe7af; }
.btn-success:hover { background: #c8f0c8; }

/* Terminal */
.term {
  background: #1e1e1e;
  border-top: 1px solid var(--border);
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  height: 220px;
}
.term-header {
  padding: 5px 12px;
  border-bottom: 1px solid #333;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: #858585;
  background: #252526;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}
.term-output {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
  font-family: var(--mono);
  font-size: 11px;
  line-height: 1.55;
  color: #cccccc;
  white-space: pre-wrap;
  word-break: break-all;
}
.term-output .line-err { color: #f48771; }
.term-output .line-sys { color: #858585; font-style: italic; }

/* Access log */
.log-list { flex: 1; overflow-y: auto; padding: 4px 0; }
.log-entry {
  padding: 6px 16px;
  border-bottom: 1px solid var(--border);
  font-size: 11px;
  cursor: default;
}
.log-entry:hover { background: var(--bg3); }
.log-entry.blocked { background: #fff0f0; }
.log-top { display: flex; justify-content: space-between; align-items: baseline; }
.log-project { font-weight: 600; color: var(--text-bright); }
.log-time { font-size: 10px; color: var(--text-dim); font-variant-numeric: tabular-nums; }
.log-bottom { display: flex; gap: 8px; margin-top: 2px; align-items: center; flex-wrap: wrap; }
.log-ip { font-family: var(--mono); color: var(--text-dim); }
.log-mac { font-family: var(--mono); font-size: 10px; color: var(--text-dim); }
.log-path { font-family: var(--mono); font-size: 10px; color: var(--text-dim); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.log-blocked-badge {
  font-size: 9px; font-weight: 800; letter-spacing: 0.06em; text-transform: uppercase;
  background: var(--red-bg); color: var(--red); padding: 1px 5px; border-radius: 3px; flex-shrink: 0;
}

/* ── MAC panel overlay (on sidebar) ── */
.side-overlay {
  position: absolute; inset: 0; background: var(--bg2);
  border-right: 1px solid var(--border);
  display: flex; flex-direction: column; z-index: 20;
}
.side-overlay.hidden { display: none; }
.overlay-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 14px; border-bottom: 1px solid var(--border); flex-shrink: 0;
  background: var(--bg3);
}
.overlay-title-label { font-size: 10px; font-weight: 700; letter-spacing: 0.1em; text-transform: uppercase; color: var(--text-dim); }
.overlay-title { font-size: 14px; font-weight: 700; color: var(--text-bright); margin-top: 2px; }
.overlay-close {
  background: none; border: none; font-size: 16px; cursor: pointer;
  color: var(--text-dim); padding: 3px 6px; border-radius: 3px; line-height: 1;
}
.overlay-close:hover { color: var(--text); background: var(--bg4); }
.overlay-body { flex: 1; overflow-y: auto; padding: 14px; display: flex; flex-direction: column; gap: 12px; }
.overlay-footer { padding: 10px 14px; border-top: 1px solid var(--border); flex-shrink: 0; }

/* MAC-specific */
.mac-toggle { display: flex; align-items: center; gap: 8px; font-size: 12px; font-weight: 500; cursor: pointer; color: var(--text); }
.mac-toggle input[type="checkbox"] { width: 14px; height: 14px; cursor: pointer; accent-color: var(--accent); }
.mac-section-label { font-size: 10px; font-weight: 700; letter-spacing: 0.1em; text-transform: uppercase; color: var(--text-dim); margin-bottom: 6px; }
.mac-list { display: flex; flex-direction: column; gap: 4px; }
.mac-item { display: flex; align-items: center; justify-content: space-between; background: var(--bg3); border: 1px solid var(--border); border-radius: 4px; padding: 6px 10px; }
.mac-addr { font-family: var(--mono); font-size: 12px; color: var(--text); }
.mac-remove { background: none; border: none; cursor: pointer; color: var(--text-dim); font-size: 10px; font-weight: 700; letter-spacing: 0.04em; text-transform: uppercase; padding: 2px 5px; border-radius: 3px; }
.mac-remove:hover { color: var(--red); background: var(--red-bg); }
.mac-empty { font-size: 11px; color: var(--text-dim); text-align: center; padding: 12px 0; }
.mac-add-row { display: flex; gap: 6px; }
.mac-hint { font-size: 10px; color: var(--text-dim); margin-top: 4px; }
.mac-hint code { font-family: var(--mono); background: var(--bg3); padding: 1px 4px; border-radius: 2px; color: var(--text); border: 1px solid var(--border); }

/* New project form */
.form-group { display: flex; flex-direction: column; gap: 4px; }
.form-label { font-size: 10px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-dim); }
.form-hint { font-size: 10px; color: var(--text-dim); }

/* Inputs */
input[type="text"], input[type="number"], textarea {
  font-family: var(--mono);
  font-size: 12px;
  background: var(--bg3);
  border: 1px solid var(--border2);
  color: var(--text);
  padding: 5px 10px;
  border-radius: 4px;
  outline: none;
  width: 100%%;
  transition: border-color 0.12s, background 0.12s;
}
input[type="text"]:focus, input[type="number"]:focus, textarea:focus {
  border-color: var(--accent);
  background: var(--bg2);
}
input[type="number"] { -moz-appearance: textfield; }

/* Save / submit button (full-width) */
.btn-save {
  width: 100%%;
  font-family: var(--font);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  padding: 8px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.12s;
}
.btn-save:hover { background: var(--accent-h); }
.btn-save:disabled { background: var(--border2); color: var(--text-dim); cursor: not-allowed; }

/* Settings tab */
.settings-pane { flex: 1; overflow-y: auto; padding: 20px 24px; display: flex; flex-direction: column; gap: 16px; }
.settings-empty { font-size: 12px; color: var(--text-dim); text-align: center; padding: 24px; }
.settings-row { display: flex; flex-direction: column; gap: 4px; }
.settings-label { font-size: 10px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-dim); }
.settings-val { font-family: var(--mono); font-size: 12px; color: var(--text); background: var(--bg3); border: 1px solid var(--border); border-radius: 4px; padding: 5px 10px; }

/* Scrollbar */
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--border2); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: #aaa; }

.empty { font-size: 12px; color: var(--text-dim); text-align: center; padding: 24px; }

/* Status bar */
.statusbar {
  height: var(--statusbar-h);
  background: var(--accent);
  display: flex;
  align-items: center;
  padding: 0 10px;
  gap: 12px;
  font-size: 11px;
  color: rgba(255,255,255,0.92);
  flex-shrink: 0;
}
.statusbar-item { display: flex; align-items: center; gap: 4px; }
.statusbar-sep { width: 1px; height: 12px; background: rgba(255,255,255,0.3); }
</style>
</head>
<body>

<!-- Titlebar -->
<div class="titlebar">
  <div class="titlebar-logo">Twisha</div>
  <div class="titlebar-sep"></div>
  <div class="titlebar-meta">Local Network Proxy</div>
  <div class="titlebar-actions">
    <button class="tb-btn" onclick="refreshAll()">↺ Refresh</button>
  </div>
</div>

<div class="layout">
  <!-- ── Sidebar ── -->
  <aside class="sidebar">
    <div class="sidebar-header">
      <span class="sidebar-section-label">Projects</span>
      <button class="btn-new-proj" onclick="showNewProjPanel()" title="New project">+</button>
    </div>
    <div class="proj-list" id="proj-list">
      %s
    </div>
    <div class="sidebar-footer">
      <div class="net-info">
        <div class="net-row"><span class="net-label">IP</span><span class="net-val">%s</span></div>
        <div class="net-row"><span class="net-label">Proxy</span><span class="net-val">:%d</span></div>
        <div class="net-row"><span class="net-label">Projects</span><span class="net-val">%d</span></div>
      </div>
    </div>

    <!-- MAC allowlist overlay -->
    <div class="side-overlay hidden" id="mac-panel">
      <div class="overlay-header">
        <div>
          <div class="overlay-title-label">MAC Allowlist</div>
          <div class="overlay-title" id="mac-panel-project"></div>
        </div>
        <button class="overlay-close" onclick="hideMACPanel()">✕</button>
      </div>
      <div class="overlay-body">
        <label class="mac-toggle">
          <input type="checkbox" id="mac-enabled" onchange="toggleMACEnabled()">
          Enable MAC filtering
        </label>
        <div id="mac-rules-body">
          <div class="mac-section-label">Allowed devices</div>
          <div class="mac-list" id="mac-list"></div>
          <div class="mac-add-row" style="margin-top:8px">
            <input type="text" id="mac-input" placeholder="aa:bb:cc:dd:ee:ff" />
            <button class="btn btn-primary" onclick="addMAC()" style="white-space:nowrap">Add</button>
          </div>
          <div class="mac-hint" style="margin-top:5px">Find MAC: <code>arp -n &lt;device-ip&gt;</code></div>
        </div>
      </div>
      <div class="overlay-footer">
        <button class="btn-save" id="btn-save-mac" onclick="saveMACs()">Save Rules</button>
      </div>
    </div>

    <!-- New project overlay -->
    <div class="side-overlay hidden" id="new-proj-panel">
      <div class="overlay-header">
        <div>
          <div class="overlay-title-label">New Project</div>
          <div class="overlay-title">Add to Twisha</div>
        </div>
        <button class="overlay-close" onclick="hideNewProjPanel()">✕</button>
      </div>
      <div class="overlay-body">
        <div class="form-group">
          <label class="form-label">Project Name *</label>
          <input type="text" id="np-name" placeholder="myapp" />
          <span class="form-hint">Accessible at <code style="font-size:10px">myapp.local</code></span>
        </div>
        <div class="form-group">
          <label class="form-label">Port *</label>
          <input type="number" id="np-port" placeholder="3000" min="1" max="65535" />
          <span class="form-hint">The port your app listens on</span>
        </div>
        <div class="form-group">
          <label class="form-label">Start Command</label>
          <input type="text" id="np-cmd" placeholder="npm run dev" />
        </div>
        <div class="form-group">
          <label class="form-label">Working Directory</label>
          <input type="text" id="np-dir" placeholder="/path/to/project" />
        </div>
      </div>
      <div class="overlay-footer" style="display:flex;flex-direction:column;gap:6px">
        <button class="btn-save" id="btn-create-proj" onclick="createProject()">Create Project</button>
        <div id="np-error" style="font-size:11px;color:var(--red);text-align:center;display:none"></div>
      </div>
    </div>
  </aside>

  <!-- ── Main panel ── -->
  <main class="main-panel">
    <div class="tab-bar">
      <button class="tab active" id="tab-overview-btn" onclick="switchTab('overview', this)">Overview</button>
      <button class="tab" id="tab-settings-btn" onclick="switchTab('settings', this)">Settings</button>
      <button class="tab" id="tab-log-btn" onclick="switchTab('log', this)">
        Access Log <span id="log-badge" class="tab-badge" style="display:none"></span>
      </button>
    </div>

    <!-- Overview tab -->
    <div class="tab-content active" id="tab-overview">
      <div class="detail-pane" id="detail-pane">
        <div class="empty" id="detail-empty">Select a project from the sidebar.</div>
        <div id="detail-content" style="display:none;flex-direction:column;gap:16px">

          <div class="detail-header">
            <div>
              <div class="detail-title" id="det-name"></div>
              <div class="detail-subtitle" id="det-sub"></div>
            </div>
            <div class="proc-controls">
              <div class="proc-status">
                <span class="proc-status-label">Status:</span>
                <span class="proc-status-val" id="det-status-label">–</span>
              </div>
              <button class="btn btn-primary" id="btn-start" onclick="startProject()">▶ Start</button>
              <button class="btn btn-danger" id="btn-stop" onclick="stopProject()" style="display:none">■ Stop</button>
            </div>
          </div>

          <!-- Stats -->
          <div class="stats-row" id="det-stats">
            <div class="stat-cell"><div class="stat-value" id="det-total">0</div><div class="stat-label">Requests</div></div>
            <div class="stat-cell"><div class="stat-value" id="det-denied" style="color:var(--red)">0</div><div class="stat-label">Blocked</div></div>
            <div class="stat-cell"><div class="stat-value" id="det-port">–</div><div class="stat-label">Port</div></div>
            <div class="stat-cell"><div class="stat-value" id="det-lat">–</div><div class="stat-label">Latency</div></div>
          </div>

          <!-- Commands card -->
          <div class="card" id="det-cmd-card">
            <div class="card-header">
              <span>Commands</span>
            </div>
            <div class="card-body" style="display:flex;flex-direction:column;gap:8px">
              <div class="settings-row">
                <div class="settings-label">Start Command</div>
                <input type="text" id="det-cmd-input" placeholder="npm run dev" />
              </div>
              <div class="settings-row">
                <div class="settings-label">Working Directory</div>
                <input type="text" id="det-dir-input" placeholder="Optional — leave blank for cwd" />
              </div>
            </div>
          </div>

        </div>
      </div>

      <!-- Terminal output -->
      <div class="term" id="term-pane" style="display:none">
        <div class="term-header">
          <span>OUTPUT — <span id="term-project-name"></span></span>
          <button class="btn btn-ghost" style="font-size:10px;padding:2px 8px;background:#333;border-color:#555;color:#aaa" onclick="clearOutput()">Clear</button>
        </div>
        <div class="term-output" id="term-output"></div>
      </div>
    </div>

    <!-- Settings tab -->
    <div class="tab-content" id="tab-settings">
      <div class="settings-pane">
        <div class="settings-empty" id="settings-empty">Select a project to view its settings.</div>
        <div id="settings-content" style="display:none;flex-direction:column;gap:16px">

          <div class="detail-header" style="margin-bottom:0">
            <div>
              <div class="detail-title" id="set-name"></div>
              <div class="detail-subtitle">Project configuration</div>
            </div>
          </div>

          <!-- Identity card -->
          <div class="card">
            <div class="card-header">Identity</div>
            <div class="card-body" style="display:flex;flex-direction:column;gap:10px">
              <div class="settings-row">
                <div class="settings-label">Project Name</div>
                <div class="settings-val" id="set-proj-name">–</div>
              </div>
              <div class="settings-row">
                <div class="settings-label">Local Domain</div>
                <div class="settings-val" id="set-domain">–</div>
              </div>
              <div class="settings-row">
                <div class="settings-label">Port</div>
                <input type="number" id="set-port" min="1" max="65535" />
              </div>
            </div>
          </div>

          <!-- Commands card -->
          <div class="card">
            <div class="card-header">Commands Structure</div>
            <div class="card-body" style="display:flex;flex-direction:column;gap:10px">
              <div class="settings-row">
                <div class="settings-label">Start Command</div>
                <input type="text" id="set-cmd" placeholder="npm run dev" />
              </div>
              <div class="settings-row">
                <div class="settings-label">Working Directory</div>
                <input type="text" id="set-dir" placeholder="Optional — leave blank for cwd" />
              </div>
            </div>
          </div>

          <!-- Save -->
          <div style="display:flex;gap:8px;align-items:center">
            <button class="btn btn-primary" id="btn-save-settings" onclick="saveSettings()">Save Settings</button>
            <span id="settings-msg" style="font-size:11px;color:var(--green);display:none">Saved</span>
          </div>

        </div>
      </div>
    </div>

    <!-- Access Log tab -->
    <div class="tab-content" id="tab-log">
      <div class="log-list" id="log-entries"><div class="empty">No requests yet.</div></div>
    </div>
  </main>
</div>

<!-- Status bar -->
<div class="statusbar">
  <div class="statusbar-item">⬤ <span id="sb-up">0</span> up</div>
  <div class="statusbar-sep"></div>
  <div class="statusbar-item">↓ <span id="sb-req">0</span> req</div>
  <div class="statusbar-sep"></div>
  <div class="statusbar-item">✗ <span id="sb-blocked">0</span> blocked</div>
  <div class="statusbar-sep"></div>
  <div class="statusbar-item" id="sb-proxy">proxy :%d</div>
</div>

<script>
var PROJECTS = %s;
var _macData = {};
var _currentProject = null;
var _currentMACs = [];
var _selectedProject = null;
var _statusData = [];
var _outputLines = [];

// ── Tab switching ──────────────────────────────────────────────────────────
function switchTab(name, btn) {
  document.querySelectorAll('.tab-content').forEach(function(el) { el.classList.remove('active'); });
  document.querySelectorAll('.tab').forEach(function(el) { el.classList.remove('active'); });
  document.getElementById('tab-' + name).classList.add('active');
  btn.classList.add('active');
}

// ── Project selection ──────────────────────────────────────────────────────
function selectProject(name) {
  _selectedProject = name;
  document.querySelectorAll('.proj-row').forEach(function(r) {
    r.classList.toggle('selected', r.dataset.name === name);
  });

  var p = PROJECTS.find(function(p) { return p.name === name; });
  if (!p) return;

  // Overview tab
  document.getElementById('detail-empty').style.display = 'none';
  var content = document.getElementById('detail-content');
  content.style.display = 'flex';
  document.getElementById('det-name').textContent = p.name;
  document.getElementById('det-sub').textContent = 'http://' + p.name + '.local';
  document.getElementById('det-port').textContent = ':' + p.port;
  document.getElementById('det-cmd-input').value = p.command || '';
  document.getElementById('det-dir-input').value = p.dir || '';
  document.getElementById('term-project-name').textContent = p.name;

  // Settings tab
  document.getElementById('settings-empty').style.display = 'none';
  var sc = document.getElementById('settings-content');
  sc.style.display = 'flex';
  document.getElementById('set-name').textContent = p.name;
  document.getElementById('set-proj-name').textContent = p.name;
  document.getElementById('set-domain').textContent = 'http://' + p.name + '.local';
  document.getElementById('set-port').value = p.port;
  document.getElementById('set-cmd').value = p.command || '';
  document.getElementById('set-dir').value = p.dir || '';
  document.getElementById('settings-msg').style.display = 'none';

  updateDetailStatus(name);
  fetchOutput(name);
}

function updateDetailStatus(name) {
  if (!name) return;
  var s = _statusData.find(function(x) { return x.name === name; });
  var reqData = _macData[name];

  if (s) {
    var isManaged = s.managed;
    var isUp = s.up;
    var statusEl = document.getElementById('det-status-label');
    if (isManaged) {
      statusEl.textContent = 'Running (managed)';
      statusEl.className = 'proc-status-val running';
      document.getElementById('btn-start').style.display = 'none';
      document.getElementById('btn-stop').style.display = '';
    } else if (isUp) {
      statusEl.textContent = 'Up (external)';
      statusEl.className = 'proc-status-val running';
      document.getElementById('btn-start').style.display = '';
      document.getElementById('btn-stop').style.display = 'none';
    } else {
      statusEl.textContent = 'Stopped';
      statusEl.className = 'proc-status-val stopped';
      document.getElementById('btn-start').style.display = '';
      document.getElementById('btn-stop').style.display = 'none';
    }
  }

  if (reqData) {
    document.getElementById('det-total').textContent = reqData.total;
    document.getElementById('det-denied').textContent = reqData.denied;
  }
}

// ── Process control ────────────────────────────────────────────────────────
function startProject() {
  if (!_selectedProject) return;
  var cmdInput = document.getElementById('det-cmd-input').value.trim();
  var dirInput = document.getElementById('det-dir-input').value.trim();
  var p = PROJECTS.find(function(x) { return x.name === _selectedProject; });
  if (p) { p.command = cmdInput; p.dir = dirInput; }

  var btn = document.getElementById('btn-start');
  btn.disabled = true;
  btn.textContent = 'Starting…';

  fetch('/api/start', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({project: _selectedProject, command: cmdInput, dir: dirInput})
  }).then(function(r) { return r.json(); }).then(function(d) {
    if (!d.ok) throw new Error(d.error || 'failed');
    document.getElementById('term-pane').style.display = 'flex';
    fetchOutput(_selectedProject);
    setTimeout(fetchAll, 500);
  }).catch(function(e) {
    alert('Start failed: ' + e.message);
  }).finally(function() {
    btn.disabled = false;
    btn.textContent = '▶ Start';
  });
}

function stopProject() {
  if (!_selectedProject) return;
  fetch('/api/stop', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({project: _selectedProject})
  }).then(function() { setTimeout(fetchAll, 500); });
}

// ── Settings save ──────────────────────────────────────────────────────────
function saveSettings() {
  if (!_selectedProject) return;
  var port = parseInt(document.getElementById('set-port').value, 10);
  var cmd  = document.getElementById('set-cmd').value.trim();
  var dir  = document.getElementById('set-dir').value.trim();
  if (!port || port < 1 || port > 65535) {
    alert('Please enter a valid port (1–65535).');
    return;
  }
  var btn = document.getElementById('btn-save-settings');
  btn.disabled = true;
  btn.textContent = 'Saving…';
  fetch('/api/update-project', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({name: _selectedProject, port: port, command: cmd, dir: dir})
  }).then(function(r) { return r.json(); }).then(function(d) {
    if (!d.ok) throw new Error(d.error || 'failed');
    // Update local PROJECTS cache
    var p = PROJECTS.find(function(x) { return x.name === _selectedProject; });
    if (p) { p.port = port; p.command = cmd; p.dir = dir; }
    // Refresh overview inputs
    document.getElementById('det-cmd-input').value = cmd;
    document.getElementById('det-dir-input').value = dir;
    document.getElementById('det-port').textContent = ':' + port;
    var msg = document.getElementById('settings-msg');
    msg.style.display = '';
    setTimeout(function() { msg.style.display = 'none'; }, 2000);
    fetchAll();
  }).catch(function(e) {
    alert('Save failed: ' + e.message);
  }).finally(function() {
    btn.disabled = false;
    btn.textContent = 'Save Settings';
  });
}

// ── Output / terminal ──────────────────────────────────────────────────────
function fetchOutput(name) {
  if (!name) return;
  fetch('/api/output?project=' + encodeURIComponent(name))
    .then(function(r) { return r.json(); })
    .then(function(lines) {
      if (!lines || lines.length === 0) return;
      document.getElementById('term-pane').style.display = 'flex';
      _outputLines = lines;
      renderOutput();
    }).catch(function() {});
}

function renderOutput() {
  var el = document.getElementById('term-output');
  el.innerHTML = _outputLines.map(function(l) {
    var cls = '';
    if (l.startsWith('▶') || l.startsWith('■')) cls = 'line-sys';
    else if (/error|Error|ERR|FAIL/i.test(l)) cls = 'line-err';
    var escaped = l.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
    return '<div class="' + cls + '">' + escaped + '</div>';
  }).join('');
  el.scrollTop = el.scrollHeight;
}

function clearOutput() {
  _outputLines = [];
  document.getElementById('term-output').innerHTML = '';
}

// ── Data fetching ──────────────────────────────────────────────────────────
function fetchAll() {
  fetchStats();
  fetchStatus();
  if (_selectedProject) fetchOutput(_selectedProject);
}

function fetchStatus() {
  fetch('/api/status').then(function(r) { return r.json(); }).then(function(data) {
    _statusData = data;
    var upCount = data.filter(function(x) { return x.up; }).length;
    document.getElementById('sb-up').textContent = upCount + '/' + data.length;

    data.forEach(function(s) {
      var row = document.getElementById('pr-' + s.name);
      if (row) {
        var dotEl = row.querySelector('.dot');
        if (dotEl) {
          dotEl.className = 'dot ' + (s.managed ? 'dot-managed' : s.up ? 'dot-up' : 'dot-down');
          dotEl.title = s.managed ? 'Managed' : s.up ? 'Up' : 'Down';
        }
        var cmdEl = document.getElementById('cmd-' + s.name);
        if (cmdEl && s.command) cmdEl.textContent = s.command;
      }
    });

    if (_selectedProject) updateDetailStatus(_selectedProject);
  }).catch(function() {});
}

function fetchStats() {
  fetch('/api/requests').then(function(r) { return r.json(); }).then(function(data) {
    _macData = {};
    var totalReq = 0, totalBlocked = 0;
    data.stats.forEach(function(s) {
      _macData[s.name] = s;
      totalReq += s.total;
      totalBlocked += s.denied;
      var shield = document.getElementById('shield-' + s.name);
      if (shield) {
        var hasRules = s.allowed_macs && s.allowed_macs.length > 0;
        shield.classList.toggle('shield-on', hasRules);
        shield.title = hasRules ? 'MAC filter: ' + s.allowed_macs.length + ' device(s)' : 'MAC Allowlist';
      }
    });
    document.getElementById('sb-req').textContent = totalReq;
    document.getElementById('sb-blocked').textContent = totalBlocked;
    updateLog(data.recent);
    if (_selectedProject) updateDetailStatus(_selectedProject);
  }).catch(function() {});
}

// ── Access log ─────────────────────────────────────────────────────────────
function reltime(d) {
  var s = Math.floor((Date.now() - new Date(d).getTime()) / 1000);
  if (s < 5) return 'now';
  if (s < 60) return s + 's ago';
  if (s < 3600) return Math.floor(s/60) + 'm ago';
  return Math.floor(s/3600) + 'h ago';
}
function trunc(s, n) { return s && s.length > n ? s.slice(0, n) + '…' : (s || ''); }

function updateLog(recent) {
  var container = document.getElementById('log-entries');
  var badge = document.getElementById('log-badge');
  if (!recent || recent.length === 0) {
    container.innerHTML = '<div class="empty">No requests yet.</div>';
    badge.style.display = 'none';
    return;
  }
  badge.style.display = '';
  badge.textContent = recent.length;
  container.innerHTML = recent.slice(0, 80).map(function(e) {
    var mac = e.mac || '—';
    var blocked = e.status === 403;
    return '<div class="log-entry' + (blocked ? ' blocked' : '') + '">' +
      '<div class="log-top"><span class="log-project">' + e.project + '</span><span class="log-time">' + reltime(e.t) + '</span></div>' +
      '<div class="log-bottom"><span class="log-ip">' + e.ip + '</span><span class="log-mac">' + mac + '</span><span class="log-path">' + trunc(e.path, 40) + '</span>' +
      (blocked ? '<span class="log-blocked-badge">blocked</span>' : '') +
      '</div></div>';
  }).join('');
}

// ── MAC panel ──────────────────────────────────────────────────────────────
function showMACPanel(project) {
  _currentProject = project;
  var s = _macData[project];
  _currentMACs = (s && s.allowed_macs) ? s.allowed_macs.slice() : [];
  document.getElementById('mac-panel-project').textContent = project;
  document.getElementById('mac-enabled').checked = _currentMACs.length > 0;
  renderMACList();
  document.getElementById('mac-panel').classList.remove('hidden');
}
function hideMACPanel() {
  document.getElementById('mac-panel').classList.add('hidden');
  _currentProject = null;
}
function toggleMACEnabled() { renderMACList(); }
function renderMACList() {
  var list = document.getElementById('mac-list');
  var body = document.getElementById('mac-rules-body');
  var enabled = document.getElementById('mac-enabled').checked;
  body.style.opacity = enabled ? '1' : '0.5';
  body.style.pointerEvents = enabled ? '' : 'none';
  if (_currentMACs.length === 0) {
    list.innerHTML = '<div class="mac-empty">No devices added yet.</div>';
    return;
  }
  list.innerHTML = _currentMACs.map(function(mac, i) {
    return '<div class="mac-item"><span class="mac-addr">' + mac + '</span>' +
      '<button class="mac-remove" onclick="removeMAC(' + i + ')">Remove</button></div>';
  }).join('');
}
function addMAC() {
  var input = document.getElementById('mac-input');
  var mac = input.value.trim().toLowerCase();
  if (!/^[0-9a-f]{2}(:[0-9a-f]{2}){5}$/.test(mac)) {
    input.style.borderColor = 'var(--red)';
    setTimeout(function() { input.style.borderColor = ''; }, 1000);
    return;
  }
  if (_currentMACs.indexOf(mac) === -1) { _currentMACs.push(mac); renderMACList(); }
  input.value = '';
}
function removeMAC(i) { _currentMACs.splice(i, 1); renderMACList(); }
function saveMACs() {
  if (!_currentProject) return;
  var enabled = document.getElementById('mac-enabled').checked;
  var macs = enabled ? _currentMACs : [];
  var btn = document.getElementById('btn-save-mac');
  btn.textContent = 'Saving…'; btn.disabled = true;
  fetch('/api/mac-rules', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({project: _currentProject, macs: macs})
  }).then(function() {
    btn.textContent = 'Saved ✓';
    fetchStats();
    setTimeout(hideMACPanel, 600);
  }).catch(function() { btn.textContent = 'Error'; })
    .finally(function() { setTimeout(function() { btn.textContent = 'Save Rules'; btn.disabled = false; }, 1400); });
}

// ── New project panel ──────────────────────────────────────────────────────
function showNewProjPanel() {
  document.getElementById('np-name').value = '';
  document.getElementById('np-port').value = '';
  document.getElementById('np-cmd').value = '';
  document.getElementById('np-dir').value = '';
  document.getElementById('np-error').style.display = 'none';
  document.getElementById('new-proj-panel').classList.remove('hidden');
  setTimeout(function() { document.getElementById('np-name').focus(); }, 50);
}
function hideNewProjPanel() {
  document.getElementById('new-proj-panel').classList.add('hidden');
}
function createProject() {
  var name = document.getElementById('np-name').value.trim();
  var port = parseInt(document.getElementById('np-port').value, 10);
  var cmd  = document.getElementById('np-cmd').value.trim();
  var dir  = document.getElementById('np-dir').value.trim();
  var errEl = document.getElementById('np-error');

  if (!name || !/^[a-z0-9-]+$/.test(name)) {
    errEl.textContent = 'Name must be lowercase letters, numbers, and hyphens only.';
    errEl.style.display = '';
    return;
  }
  if (!port || port < 1 || port > 65535) {
    errEl.textContent = 'Please enter a valid port (1–65535).';
    errEl.style.display = '';
    return;
  }
  errEl.style.display = 'none';
  var btn = document.getElementById('btn-create-proj');
  btn.disabled = true; btn.textContent = 'Creating…';

  fetch('/api/projects', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({name: name, port: port, command: cmd, dir: dir})
  }).then(function(r) { return r.json(); }).then(function(d) {
    if (!d.ok) throw new Error(d.error || 'failed');
    // Add to local PROJECTS list and rebuild sidebar entry
    PROJECTS.push({name: name, port: port, command: cmd, dir: dir});
    var list = document.getElementById('proj-list');
    var div = document.createElement('div');
    div.className = 'proj-row';
    div.id = 'pr-' + name;
    div.dataset.name = name;
    div.setAttribute('onclick', "selectProject('" + name + "')");
    div.innerHTML =
      '<div class="proj-left">' +
        '<span class="dot dot-down" title="Down"></span>' +
        '<div class="proj-meta">' +
          '<div class="proj-name">' + name + '</div>' +
          '<div class="proj-cmd" id="cmd-' + name + '">' + (cmd || 'No command') + '</div>' +
        '</div>' +
      '</div>' +
      '<div class="proj-right">' +
        '<div class="proj-port">:' + port + '</div>' +
        '<button class="btn-shield" id="shield-' + name + '" onclick="event.stopPropagation();showMACPanel(\'' + name + '\')" title="MAC Allowlist">' +
          '<svg width="11" height="13" viewBox="0 0 12 14" fill="none"><path d="M6 0.5L0.5 2.5V7C0.5 10.05 2.9 12.9 6 13.85C9.1 12.9 11.5 10.05 11.5 7V2.5L6 0.5Z" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"/></svg>' +
        '</button>' +
      '</div>';
    list.appendChild(div);
    hideNewProjPanel();
    selectProject(name);
    fetchAll();
  }).catch(function(e) {
    errEl.textContent = e.message;
    errEl.style.display = '';
  }).finally(function() {
    btn.disabled = false; btn.textContent = 'Create Project';
  });
}

function refreshAll() { fetchAll(); }

// ── Boot ───────────────────────────────────────────────────────────────────
fetchAll();
setInterval(fetchAll, 4000);
if (PROJECTS && PROJECTS.length > 0) {
  setTimeout(function() { selectProject(PROJECTS[0].name); }, 100);
}
</script>
</body>
</html>`,
		projectRows.String(),
		ip, cfg.ProxyPort, len(cfg.Projects),
		cfg.ProxyPort,
		string(projectsJSON))
}
