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
slurp is ready
Saving to: ~/Downloads/slurp

Push a file:
  curl -T photo.jpg "https://xyz.trycloudflare.com/upload?token=abc123de&filename=photo.jpg"

Ctrl+C to stop.
```

## Options

| Flag          | Env            | Default               | Description                      |
|---------------|----------------|-----------------------|----------------------------------|
| `--port`      | `PORT`         | First free from 8765  | Listen port                      |
| `--dir`       | `UPLOAD_DIR`   | `~/Downloads/slurp`   | Directory to save uploaded files |
| `--token`     | `UPLOAD_TOKEN` | Auto-generated        | Auth token                       |
| `--no-tunnel` | —              | false                 | Disable cloudflared tunnel       |

## Requirements

- [cloudflared](https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/) (optional — for public URL)
