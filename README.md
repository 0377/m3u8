# M3U8

[![CI](https://github.com/0377/m3u8/actions/workflows/ci.yml/badge.svg)](https://github.com/0377/m3u8/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/github/license/0377/m3u8)](LICENSE)
[![Release](https://img.shields.io/github/v/release/0377/m3u8)](https://github.com/0377/m3u8/releases)

A lightweight M3U8 downloader written in Go. It parses HLS playlists, downloads TS segments concurrently, decrypts AES-128 encrypted streams, merges them into a single file, and optionally remuxes to MP4 via ffmpeg.

Two modes are available:

- **CLI** — one-shot download from the command line
- **HTTP API** — async task server with progress polling and file download

[中文说明](README_zh-CN.md)

## Features

- Parse and download VOD M3U8 playlists
- Auto-resolve Master playlists (selects the first variant)
- Concurrent TS segment download with configurable workers
- Per-segment retry on failure
- AES-128 segment decryption
- Merge TS segments into a single file
- Remux to MP4 via ffmpeg (stream copy, no re-encoding)
- Custom HTTP headers, Cookie, and optional TLS skip-verify
- HTTP API server for remote parsing, async downloads, progress tracking, and cancellation
- Optional API Key authentication, CORS, task TTL, and automatic cleanup

## Requirements

- Go 1.22+ (for building from source)
- [ffmpeg](https://ffmpeg.org/) (optional, required when MP4 output is enabled — enabled by default)

## Quick Start

```bash
make vendor && make build
./m3u8 -u=https://example.com/index.m3u8 -o=./output
```

Or run from source:

```bash
go run -mod=vendor . -u=https://example.com/index.m3u8 -o=./output
```

## CLI Usage

```bash
m3u8 -u <URL> [options]
m3u8 serve [options]          # start HTTP API server
```

### Examples

```bash
# Basic download (outputs ./main.mp4 by default)
./m3u8 -u=https://example.com/index.m3u8

# Specify output directory and filename
./m3u8 -u=https://example.com/index.m3u8 -o=./output -f myvideo

# Keep TS output only (skip MP4 conversion)
./m3u8 -u=https://example.com/index.m3u8 -mp4=false

# Sites requiring Referer or Cookie
./m3u8 -u=https://example.com/index.m3u8 \
  -H "Referer: https://example.com/" \
  -cookie "session=abc"

# Self-signed HTTPS certificate
./m3u8 -u=https://self-signed.example.com/index.m3u8 -k
```

### Platform-specific

Linux & macOS:

```bash
./m3u8 -u=https://example.com/index.m3u8 -o=./output
```

Windows PowerShell:

```powershell
.\m3u8.exe -u="https://example.com/index.m3u8" -o="D:\data\output"
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-u` | | M3U8 URL (**required**) |
| `-o` | `.` | Output directory |
| `-f` | `main` | Output base name (`.ts` / `.mp4` extension optional) |
| `-c` | `25` | Download concurrency |
| `-r` | `10` | Max retries per segment |
| `-mp4` | `true` | Remux merged TS to MP4 via ffmpeg (`-mp4=false` to disable) |
| `-H` | | Custom HTTP header (`"Key: Value"`), repeatable |
| `-cookie` | | Cookie request header |
| `-k` | `false` | Skip HTTPS certificate verification (insecure) |
| `-h` | | Show help |

> VOD playlists only. Some sources rate-limit requests — lower `-c` or raise `-r` as needed.

## HTTP API

Run `m3u8 serve` to start an HTTP API server for parsing playlists, creating async download tasks, polling progress, and downloading finished files.

### Starting the Server

**Development (no auth)**

```bash
make build
./m3u8 serve --port 8080 --data-dir ./data
```

**Production (with API Key auth)**

```bash
./m3u8 serve \
  --port 8080 \
  --data-dir /var/m3u8/data \
  --auth-enabled \
  --api-key "your-secret-key" \
  --max-tasks 3 \
  --task-ttl 24h
```

### Server Options

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8080` | Listen port |
| `--data-dir` | `./data` | Task and output storage directory |
| `--auth-enabled` | `false` | Enable API Key authentication |
| `--api-key` | | API Key (required when `--auth-enabled`) |
| `--cors-origins` | `*` | Allowed CORS origins (comma-separated) |
| `--max-tasks` | `3` | Max concurrent download tasks |
| `--task-ttl` | `24h` | Retention period for completed tasks |
| `--cleanup-interval` | `1h` | Expired task cleanup interval |

When auth is enabled, all endpoints except health check require `X-API-Key: <key>` or `Authorization: Bearer <key>`.

### Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v1/health` | No | Health check |
| `POST` | `/api/v1/parse` | Yes* | Parse M3U8 playlist |
| `POST` | `/api/v1/tasks` | Yes* | Create download task |
| `GET` | `/api/v1/tasks` | Yes* | List tasks (`status`, `limit`, `offset` query params) |
| `GET` | `/api/v1/tasks/{taskID}` | Yes* | Get task status and progress |
| `GET` | `/api/v1/tasks/{taskID}/download` | Yes* | Download completed task output |
| `DELETE` | `/api/v1/tasks/{taskID}` | Yes* | Cancel a running task |

\* Required only when `--auth-enabled` is set.

### Task Status

`pending` → `running` → `completed` | `failed` | `cancelled` | `expired`

### API Examples

Parse an M3U8 URL (returns first 5 segments by default; use `?full=true` for all):

```bash
curl -X POST http://localhost:8080/api/v1/parse \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/index.m3u8"}'
```

Create a task and poll until complete:

```bash
# Create task
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/index.m3u8","filename":"myvideo","concurrency":25,"to_mp4":true}'

# Poll status (replace <taskID> with returned task_id)
curl http://localhost:8080/api/v1/tasks/<taskID>

# Download output (when status is completed)
curl -OJ http://localhost:8080/api/v1/tasks/<taskID>/download

# Cancel a running task
curl -X DELETE http://localhost:8080/api/v1/tasks/<taskID>
```

When auth is enabled, add `-H "X-API-Key: your-secret-key"` to the requests above.

### Create Task Request Body

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `url` | string | | M3U8 URL (**required**) |
| `filename` | string | `main` | Output base name |
| `concurrency` | int | `25` | Download workers |
| `to_mp4` | bool | `true` | Remux to MP4 after merge |

## Development

Go Modules + vendor mode (`-mod=vendor`) for reproducible offline builds.

```bash
make vendor   # populate vendor/
make build    # build binary
make test     # run tests
```

Cross-compile:

```bash
make build-linux
make build-darwin-arm64
make build-windows
```

## Download

[Binary packages](https://github.com/0377/m3u8/releases)

[Upstream releases](https://github.com/oopsguy/m3u8/releases)

## References

- [TS科普 2 包头](https://blog.csdn.net/cabbage2008/article/details/49281729)
- [HTTP Live Streaming draft-pantos-http-live-streaming-23](https://tools.ietf.org/html/draft-pantos-http-live-streaming-23#section-4.3.4.2)
- [MPEG transport stream - Wikipedia](https://en.wikipedia.org/wiki/MPEG_transport_stream)

## License

[MIT License](./LICENSE)
