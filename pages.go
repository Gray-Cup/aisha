package main

import (
	"fmt"
	"net/http/httputil"
	"strings"
)

// ─────────────────────────────────────────────
//  Error pages (light mode)
// ─────────────────────────────────────────────

func forbiddenPage(project, clientIP, mac, serverIP string, adminPort int) string {
	macStr := mac
	if macStr == "" {
		macStr = "unknown (not in ARP table)"
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Twisha — Access Denied</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Helvetica Neue', sans-serif;
    background: #f3f3f3;
    display: flex; align-items: center; justify-content: center;
    min-height: 100vh;
    -webkit-font-smoothing: antialiased;
    color: #383838;
  }
  .card {
    max-width: 460px; width: 100%%;
    background: #ffffff;
    border: 1px solid #e0e0e0;
    border-radius: 8px;
    padding: 2rem;
    box-shadow: 0 1px 4px rgba(0,0,0,0.08);
  }
  .badge {
    display: inline-block;
    font-size: .6875rem; font-weight: 700; letter-spacing: .1em; text-transform: uppercase;
    background: #fce8e8; color: #c5271f;
    border: 1px solid #e8b4b4;
    padding: 2px 8px; border-radius: 4px;
    margin-bottom: .75rem;
  }
  h1 { font-size: 1.125rem; font-weight: 700; color: #1a1a1a; letter-spacing: -.02em; margin-bottom: 1.25rem; }
  .row {
    display: flex; justify-content: space-between; align-items: center;
    padding: .5rem 0; border-bottom: 1px solid #f0f0f0; font-size: .875rem;
  }
  .row:last-of-type { border-bottom: none; }
  .row-label { color: #717171; }
  .row-val { font-family: 'SF Mono', ui-monospace, monospace; font-size: .8125rem; font-weight: 600; color: #1a1a1a; }
  .btn {
    display: inline-block; margin-top: 1.5rem;
    font-size: .75rem; font-weight: 700; letter-spacing: .06em; text-transform: uppercase;
    padding: .5rem 1.125rem;
    background: #005fb8; color: #fff;
    border-radius: 4px; text-decoration: none;
    transition: background .12s;
  }
  .btn:hover { background: #004c99; }
</style>
</head>
<body>
<div class="card">
  <div class="badge">403 — Access Denied</div>
  <h1>Your device is not on the allowlist for this project.</h1>
  <div class="row"><span class="row-label">Your IP</span><span class="row-val">%s</span></div>
  <div class="row"><span class="row-label">Your MAC</span><span class="row-val">%s</span></div>
  <div class="row"><span class="row-label">Project</span><span class="row-val">%s</span></div>
  <a class="btn" href="http://%s:%d">Manage Access →</a>
</div>
</body>
</html>`, clientIP, macStr, project, serverIP, adminPort)
}

func notFoundPage(host string, routes map[string]*httputil.ReverseProxy, ip string) string {
	var links strings.Builder
	for k := range routes {
		links.WriteString(fmt.Sprintf(
			`<li><a href="http://%s">%s</a></li>`, k, k,
		))
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Twisha — Not Found</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Helvetica Neue', sans-serif;
    background: #f3f3f3;
    display: flex; align-items: center; justify-content: center;
    min-height: 100vh;
    -webkit-font-smoothing: antialiased;
    color: #383838;
  }
  .card {
    max-width: 440px; width: 100%%;
    background: #ffffff;
    border: 1px solid #e0e0e0;
    border-radius: 8px;
    padding: 2rem;
    box-shadow: 0 1px 4px rgba(0,0,0,0.08);
  }
  .badge {
    display: inline-block;
    font-size: .6875rem; font-weight: 700; letter-spacing: .1em; text-transform: uppercase;
    background: #f0f0f0; color: #717171;
    border: 1px solid #e0e0e0;
    padding: 2px 8px; border-radius: 4px;
    margin-bottom: .75rem;
  }
  h1 { font-size: 1.125rem; font-weight: 700; color: #1a1a1a; letter-spacing: -.02em; margin-bottom: 1rem; }
  p { font-size: .875rem; color: #717171; margin: .5rem 0; }
  ul { margin: .75rem 0 0 1.25rem; }
  li { margin: .3rem 0; font-size: .875rem; }
  a { color: #005fb8; text-decoration: none; }
  a:hover { text-decoration: underline; }
  .btn {
    display: inline-block; margin-top: 1.25rem;
    font-size: .75rem; font-weight: 700; letter-spacing: .05em; text-transform: uppercase;
    padding: .5rem 1rem;
    background: #005fb8; color: #fff;
    border-radius: 4px; text-decoration: none;
    transition: background .12s;
  }
  .btn:hover { background: #004c99; }
</style>
</head>
<body>
<div class="card">
  <div class="badge">404 — Unknown Host</div>
  <h1>%s</h1>
  <p>This domain isn't registered in your Twisha config.</p>
  <p><strong style="color:#1a1a1a">Available projects:</strong></p>
  <ul>%s</ul>
  <a class="btn" href="http://%s:9090">Open Dashboard →</a>
</div>
</body>
</html>`, host, links.String(), ip)
}
