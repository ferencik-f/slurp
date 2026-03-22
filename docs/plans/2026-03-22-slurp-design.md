# slurp — Design Document
*2026-03-22*

## Overview

`slurp` is a minimal, zero-config Go binary that lets you securely push files from any machine to your local machine via a public tunnel. Run one command, get a copy-pasteable `curl` one-liner, done.

---

## Goals

- Single binary, no config file required
- Smart defaults — works with `slurp` and no arguments
- Auto-launches cloudflared tunnel if available
- Prints ready-to-use curl command on startup
- Accepts files from any machine via `curl -T`

---

## Project Structure

```
slurp/
  main.go
  README.md
  .github/
    workflows/
      release.yml     ← goreleaser for cross-platform binaries
```

No external dependencies — stdlib only.

---

## Endpoints

| Method   | Path      | Description              |
|----------|-----------|--------------------------|
| GET      | /health   | Returns 200 OK           |
| POST/PUT | /upload   | Receive a file (raw body)|

---

## Authentication

Token is checked in this order:
1. `Authorization: Bearer <token>` header
2. `?token=<token>` query param

Both are supported equally.

---

## Filename Resolution

Filename is optional. Resolved in this order:
1. `?filename=` query param
2. `X-Filename` request header
3. Fallback: `upload-20060102-150405.bin` (timestamp — no collisions)

**Collision handling:** If the resolved filename already exists on disk, a numeric suffix is appended before the extension — e.g., `foo.txt` → `foo (1).txt` → `foo (2).txt`. Applied to all resolution paths including the timestamp fallback.

---

## Configuration

All flags have environment variable equivalents. All have smart defaults.

| Flag        | Env          | Default                  | Description             |
|-------------|--------------|--------------------------|-------------------------|
| `--port`    | `PORT`       | First free port from 8765 | Listen port             |
| `--dir`     | `UPLOAD_DIR` | `~/Downloads/slurp`      | Directory to save files (auto-created if missing) |
| `--token`   | `UPLOAD_TOKEN`| Auto-generated (16 chars)| Auth token              |
| `--no-tunnel` | —          | false                    | Disable cloudflared     |

---

## Tunnel Integration

On startup, `slurp`:
1. Starts the HTTP server
2. If `cloudflared` is in PATH, spawns `cloudflared tunnel --url http://localhost:PORT`
3. Parses stdout for the public URL (e.g. `https://xyz.trycloudflare.com`)
4. Once URL is known, prints the startup banner

If `cloudflared` is not installed, prints a friendly install hint and falls back to local-only mode.

---

## Startup Output

```
slurp is ready  ✓
Saving to: ~/Downloads/slurp

Push a file from anywhere:

  curl -T <file> "https://xyz.trycloudflare.com/upload?token=abc123de&filename=<file>"

Or with header auth:

  curl -T <file> -H "Authorization: Bearer abc123de" \
    "https://xyz.trycloudflare.com/upload?filename=<file>"

Ctrl+C to stop.
```

---

## Shutdown Behavior

- **Ctrl+C once** (no active uploads): clean shutdown immediately
- **Ctrl+C once** (upload in progress): print `Upload in progress — press Ctrl+C again to force quit` and keep running
- **Ctrl+C twice**: force shutdown, even if an upload is mid-transfer

---

## Example Usage

```bash
# Run with zero config
slurp

# Custom dir and token
slurp --dir ~/received --token mysecret

# Local only (no tunnel)
slurp --no-tunnel
```

---

## Non-Goals (v1)

- Web UI (planned for later)
- Multi-file / zip bundling
- Progress bar on receiver side
- Authentication beyond shared token
