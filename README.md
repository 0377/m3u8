# M3U8

M3U8 - a mini M3U8 downloader written in Golang for downloading and merging TS(Transport Stream) files.

You only need to specify the flags(`u`, `o`, `c`) to run, downloader will automatically download all TS files and consolidate them into a single TS file.

[中文说明](README_zh-CN.md)

## Features

- Download and parse M3U8（VOD）
- Retry on download TS failure
- Parse Master playlist
- Decrypt TS
- Merge TS

## Usage

```bash
make vendor && make build
./m3u8 -u=http://example.com/index.m3u8 -o=/data/example
```

Or run from source:

```bash
go run -mod=vendor . -u=http://example.com/index.m3u8 -o=/data/example
```

### binary

Linux & MacOS

```
./m3u8 -u=http://example.com/index.m3u8 -o=/data/example
```

Windows PowerShell

```
.\m3u8.exe -u="http://example.com/index.m3u8" -o="D:\data\example"
```

Flags:

```
-u  M3U8 URL (required)
-o  Output folder (required)
-c  Concurrency, default 25
```

## HTTP API

In addition to CLI download, run `m3u8 serve` to start an HTTP API server for parsing M3U8 playlists, creating async download tasks, polling progress, and downloading finished files.

### Starting the server

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

Server options:

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

### Examples

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
  -d '{"url":"https://example.com/index.m3u8","filename":"myvideo","concurrency":25}'

# Poll status (replace <taskID> with returned task_id)
curl http://localhost:8080/api/v1/tasks/<taskID>

# Download output (when status is completed)
curl -OJ http://localhost:8080/api/v1/tasks/<taskID>/download
```

When auth is enabled, add `-H "X-API-Key: your-secret-key"` to the requests above.

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
## Screenshots

![Demo](./screenshots/demo.gif)

## References

- [TS科普 2 包头](https://blog.csdn.net/cabbage2008/article/details/49281729)
- [HTTP Live Streaming draft-pantos-http-live-streaming-23](https://tools.ietf.org/html/draft-pantos-http-live-streaming-23#section-4.3.4.2)
- [MPEG transport stream - Wikipedia](https://en.wikipedia.org/wiki/MPEG_transport_stream)


## License

[MIT License](./LICENSE)