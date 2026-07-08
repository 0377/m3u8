# M3U8

M3U8 是一个使用了 Go 语言编写的迷你 M3U8 下载工具。你只需指定必要的 flag (`u`、`o`、`c`) 来运行, 工具就会自动帮你解析 M3U8 文件，并将 TS 片段下载下来合并成一个文件。


## 功能

- 下载和解析 M3U8（仅限 VOD 类型）
- 下载 TS 失败重试
- 解析 Master playlist
- 解密 TS（内置 AES-128）
- 解密脚本支持（Starlark / 外部进程，配置文件 `decrypt.yaml`）
- 合并 TS 片段

## 用法

### 开发构建（vendor 模式）

项目使用 Go Modules + vendor 模式，构建时强制从 `vendor/` 目录读取依赖，保证可复现。

```bash
# 安装依赖到 vendor/（首次或更新依赖后）
make vendor

# 编译
make build

# 运行测试
make test
```

也可直接：

```bash
GOFLAGS=-mod=vendor go build -o m3u8 .
```

### 源码方式

```bash
go run -mod=vendor . -u=http://example.com/index.m3u8 -o=/data/example
```

### 二进制方式:

Linux 和 MacOS

```
./m3u8 -u=http://example.com/index.m3u8 -o=/data/example
```

Windows PowerShell

```
.\m3u8.exe -u="http://example.com/index.m3u8" -o="D:\data\example"
```

参数说明：

```
- u M3U8 地址
- o 文件保存目录
- c 下载协程并发数，默认 25
```

部分链接可能限制请求频率，可根据实际情况调整 `c` 参数的值。

## HTTP API

除 CLI 下载外，还可通过 `m3u8 serve` 启动 HTTP API 服务，支持解析 M3U8、创建异步下载任务、查询进度与下载成品文件。

### 启动服务

**开发环境（无认证）**

```bash
make build
./m3u8 serve --port 8080 --data-dir ./data
```

**生产环境（启用 API Key 认证）**

```bash
./m3u8 serve \
  --port 8080 \
  --data-dir /var/m3u8/data \
  --auth-enabled \
  --api-key "your-secret-key" \
  --max-tasks 3 \
  --task-ttl 24h
```

服务选项：

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `--port` | `8080` | 监听端口 |
| `--data-dir` | `./data` | 任务与输出文件存储目录 |
| `--auth-enabled` | `false` | 是否启用 API Key 认证 |
| `--api-key` | | API Key（`--auth-enabled` 时必填） |
| `--cors-origins` | `*` | CORS 允许来源（逗号分隔） |
| `--max-tasks` | `3` | 最大并发下载任务数 |
| `--task-ttl` | `24h` | 已完成任务保留时长 |
| `--cleanup-interval` | `1h` | 过期任务清理间隔 |

启用认证后，除健康检查外的接口需在请求头携带 `X-API-Key: <key>` 或 `Authorization: Bearer <key>`。

### 接口一览

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| `GET` | `/api/v1/health` | 否 | 健康检查 |
| `POST` | `/api/v1/parse` | 是* | 解析 M3U8 播放列表 |
| `POST` | `/api/v1/tasks` | 是* | 创建下载任务 |
| `GET` | `/api/v1/tasks` | 是* | 列出任务（支持 `status`、`limit`、`offset` 查询参数） |
| `GET` | `/api/v1/tasks/{taskID}` | 是* | 查询单个任务状态与进度 |
| `GET` | `/api/v1/tasks/{taskID}/download` | 是* | 下载已完成任务的输出文件 |
| `DELETE` | `/api/v1/tasks/{taskID}` | 是* | 取消进行中的任务 |

\* 仅当 `--auth-enabled` 时要求认证。

### 使用示例

解析 M3U8（默认返回前 5 个分片，加 `?full=true` 返回全部）：

```bash
curl -X POST http://localhost:8080/api/v1/parse \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/index.m3u8"}'
```

创建下载任务并轮询状态：

```bash
# 创建任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/index.m3u8","filename":"myvideo","concurrency":25}'

# 查询任务（将 <taskID> 替换为返回的 task_id）
curl http://localhost:8080/api/v1/tasks/<taskID>

# 下载成品（任务 status 为 completed 后）
curl -OJ http://localhost:8080/api/v1/tasks/<taskID>/download
```

启用认证时，在上述请求中增加 `-H "X-API-Key: your-secret-key"`。

## 下载

[二进制文件](https://github.com/0377/m3u8/releases)

## 截屏

![Demo](./screenshots/demo.gif)

## 参考资料

- [TS科普 2 包头](https://blog.csdn.net/cabbage2008/article/details/49281729)
- [HTTP Live Streaming draft-pantos-http-live-streaming-23](https://tools.ietf.org/html/draft-pantos-http-live-streaming-23#section-4.3.4.2)
- [MPEG transport stream - Wikipedia](https://en.wikipedia.org/wiki/MPEG_transport_stream)


## License

[MIT License](./LICENSE)