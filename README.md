# slurp

Push files from any machine to yours with a single `curl` command.

## Install

Build from source:

```bash
go install github.com/feroferencik/slurp@latest
```

Or download a pre-built binary from [Releases](../../releases).

## Usage

```bash
# Zero config — auto-generates token, finds free port, starts cloudflared tunnel
slurp

# Custom options
slurp --dir ~/received --token mysecret --port 9000

# Local only (no tunnel)
slurp --no-tunnel
```

On startup, slurp prints a ready-to-use `curl` command:

```
slurp  ·  ready
dir    ~/Downloads/slurp
token  abc123...

curl -T photo.jpg -H "Authorization: Bearer abc123..." \
  "https://xyz.trycloudflare.com/upload/photo.jpg"

Ctrl+C to stop.
```

Query-string auth remains supported for compatibility, but bearer auth is the preferred default because it is less likely to leak through shell history, logs, or screenshots.

## Options

| Flag          | Preferred Env                  | Default               | Description                      |
|---------------|--------------------------------|-----------------------|----------------------------------|
| `--port`      | `SLURP_PORT` (`PORT` legacy)   | First free from 8765  | Listen port                      |
| `--dir`       | `SLURP_DIR` (`UPLOAD_DIR` legacy) | `~/Downloads/slurp`   | Directory to save uploaded files |
| `--token`     | `SLURP_TOKEN` (`UPLOAD_TOKEN` legacy) | Auto-generated        | Auth token                       |
| `--no-tunnel` | —                              | false                 | Disable cloudflared tunnel       |

## Requirements

- [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/) (optional — for public URL)
