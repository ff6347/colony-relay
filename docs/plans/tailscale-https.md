# Tailscale HTTPS proxy on start

## Problem

The relay serves plain HTTP. Users accessing the web UI over Tailscale get `http://hostname:4100` but Tailscale can provide automatic HTTPS with provisioned TLS certs via `tailscale serve`.

## Approach

Integrate `tailscale serve` into the `start` command so HTTPS is available automatically when Tailscale is installed and connected.

### On start

1. Check if `tailscale` CLI is available (already done for hostname detection)
2. Check if Tailscale is connected (`tailscale status --json` → `BackendState: "Running"`)
3. Run `tailscale serve --bg <port>` to set up HTTPS reverse proxy
4. Show `https://hostname.tailnet.ts.net` in the address output (no port needed, defaults to 443)

### On shutdown

Run `tailscale serve --https=443 off` to remove only our proxy. Do NOT use `tailscale serve reset` as that clears the entire serve config and would nuke unrelated user proxies.

### Idempotency

Before configuring, check `tailscale serve status --json` to see if a serve config already exists for port 443. If it points to our relay port, skip. If it points to something else, do not overwrite — log a warning and skip the HTTPS setup.

## Files to modify

- `pkg/discover/addresses.go` — add `SetupTailscaleServe(port int) error` and `TeardownTailscaleServe() error`
- `cmd/colony-relay/start.go` — call setup after listener is ready, call teardown in cleanup
- `pkg/discover/addresses.go` — `ListenAddresses` should detect active tailscale serve and show `https://` URL

## Edge cases

- Tailscale not installed: skip silently
- Tailscale installed but not connected: skip silently
- Port 443 already used by another tailscale serve: warn and skip
- `tailscale serve` command fails: warn and continue with HTTP only
- Shutdown cleanup fails: log warning, don't block exit

## References

- `tailscale serve --bg <port>` — background HTTPS proxy, persists across reboots
- `tailscale serve --https=443 off` — remove specific proxy
- `tailscale serve status --json` — check current config
- `tailscale serve reset` — clear all (avoid this)
- TLS certs are provisioned automatically by Tailscale
