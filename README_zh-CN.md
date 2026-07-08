# M3U8

[![CI](https://github.com/0377/m3u8/actions/workflows/ci.yml/badge.svg)](https://github.com/0377/m3u8/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/github/license/0377/m3u8)](LICENSE)
[![Release](https://img.shields.io/github/v/release/0377/m3u8)](https://github.com/0377/m3u8/releases)

基于 Go 的轻量级 M3U8 下载工具（v1.3.0）。支持解析 HLS 播放列表、并发下载 TS 分片、AES-128 与云点播加密解密、合并为单个文件，并可通过 ffmpeg 封装为 MP4。

提供两种使用方式：

- **CLI 命令行** — 一次性本地下载
- **HTTP API 服务** — 远程解析、异步任务、进度查询与文件下载

[English](README.md)

## 功能

- 解析并下载 VOD 类型 M3U8 播放列表
- 自动解析 Master playlist（选取第一个变体流）
- 可配置并发数的 TS 分片下载
- 单分片失败自动重试
- 分片级断点续传（相同 `-u`、`-o`、`-f` 重跑自动跳过已完成分片）
- AES-128 分片解密
- 内置云点播解密（腾讯云 SimpleAES、阿里云 HLS 标准加密）
- 解密脚本支持（Starlark / 外部进程，配置文件 `decrypt.yaml`）
- 合并 TS 分片为单个文件
- 通过 ffmpeg 封装为 MP4（流复制，不重新编码）
- 下载与合并过程单行进度条显示
- 自定义 HTTP 请求头、Cookie，可选跳过 TLS 证书验证
- HTTP/HTTPS 代理支持（CLI `-proxy` 或 `HTTP_PROXY` / `HTTPS_PROXY` 环境变量）
- HTTP API 服务：远程解析、异步下载、进度跟踪、任务取消
- 可选 API Key 认证、CORS、任务 TTL 与自动清理

## 环境要求

- Go 1.22+（源码构建）
- [ffmpeg](https://ffmpeg.org/)（可选，启用 MP4 输出时需要 — 默认开启）

## 快速开始

```bash
make vendor && make build
./m3u8 -u=https://example.com/index.m3u8 -o=./output
```

或直接运行源码：

```bash
go run -mod=vendor . -u=https://example.com/index.m3u8 -o=./output
```

## CLI 用法

```bash
m3u8 -u <URL> [选项]
m3u8 serve [选项]          # 启动 HTTP API 服务
```

### 示例

```bash
# 基本下载（默认输出 ./main.mp4）
./m3u8 -u=https://example.com/index.m3u8

# 指定输出目录和文件名
./m3u8 -u=https://example.com/index.m3u8 -o=./output -f myvideo

# 仅保留 TS 文件（不转 MP4）
./m3u8 -u=https://example.com/index.m3u8 -mp4=false

# 需要 Referer 或 Cookie 的站点
./m3u8 -u=https://example.com/index.m3u8 \
  -H "Referer: https://example.com/" \
  -cookie "session=abc"

# 自签名 HTTPS 证书
./m3u8 -u=https://self-signed.example.com/index.m3u8 -k

# 通过 HTTP 代理下载
./m3u8 -u=https://example.com/index.m3u8 -proxy http://127.0.0.1:7890

# 自定义解密脚本
./m3u8 -u=https://example.com/index.m3u8 -decrypt-script scripts/custom.star

# 断点续传（使用相同的 -u、-o、-f）
./m3u8 -u=https://example.com/index.m3u8 -o=./output -f myvideo
```

### 各平台运行

Linux 和 macOS：

```bash
./m3u8 -u=https://example.com/index.m3u8 -o=./output
```

Windows PowerShell：

```powershell
.\m3u8.exe -u="https://example.com/index.m3u8" -o="D:\data\output"
```

### CLI 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-u` | | M3U8 地址（**必填**） |
| `-o` | `.` | 输出目录 |
| `-f` | `main` | 输出文件名（可带 `.ts` / `.mp4` 扩展名） |
| `-c` | `25` | 下载并发数 |
| `-r` | `10` | 单分片最大重试次数 |
| `-mp4` | `true` | 合并后转 MP4（`-mp4=false` 关闭） |
| `-H` | | 自定义 HTTP 请求头（`"Key: Value"`），可重复指定 |
| `-cookie` | | Cookie 请求头 |
| `-proxy` | | HTTP 代理地址（如 `http://127.0.0.1:7890`） |
| `-k` | `false` | 跳过 HTTPS 证书验证（不安全） |
| `-decrypt-script` | | 解密脚本路径（`.star` 或 `.py`） |
| `-decrypt-config` | `decrypt.yaml` | 解密配置文件路径 |
| `-scripts-dir` | `scripts` | 解密脚本库目录 |
| `-drm-token` | | 腾讯云 DrmToken（SimpleAES） |
| `-pkey` | | 腾讯云 SimpleAES 播放密钥 |
| `-mts-token` | | 阿里云 MtsHlsUriToken |
| `-h` | | 显示帮助信息 |

> 仅支持 VOD 类型。部分链接限制请求频率，可适当调低 `-c` 或提高 `-r`。
>
> 未指定 `-proxy` 时，自动读取 `HTTP_PROXY` / `HTTPS_PROXY` 环境变量。
>
> 中断后使用相同的 `-u`、`-o`、`-f` 重新运行即可续传，`ts/` 中已完成的分片会自动复用。

## 云点播加密

工具内置腾讯云 SimpleAES 与阿里云 HLS 标准加密的 URL 预处理与 Key 二次解密。检测到云点播特征时自动启用内置 Provider，**通常无需** `-decrypt-script`。

| Provider | 自动检测（满足其一） | Key 处理 |
|----------|---------------------|----------|
| 腾讯云 SimpleAES | M3U8 域名为 `*.vod2.myqcloud.com`；或 key URI 含 `drm.vod2.myqcloud.com` 且 `drmType=SimpleAES` | `SHA256(pkey)` → AES-CBC（零 IV）解密密文 key |
| 阿里云 HLS 标准加密 | M3U8 URL 含 `MtsHlsUriToken=`；或 key URI 含 `Ciphertext=` | 响应为 16 字节二进制或 Base64 文本 → 16 字节 AES key |

凭证通过 CLI 参数手动传入，不写入配置文件或 `decrypt.yaml`。

| 参数 | 说明 |
|------|------|
| `-drm-token` | 腾讯云 DrmToken（在 M3U8 路径中插入 `voddrm.token.{token}`） |
| `-pkey` | 腾讯云 SimpleAES 播放密钥 |
| `-mts-token` | 阿里云 `MtsHlsUriToken`（拼接到 M3U8 URL 查询参数） |

```bash
# 腾讯云 SimpleAES
./m3u8 -u "https://1500014561.vod2.myqcloud.com/.../adp.12.m3u8?t=...&sign=..." \
  -drm-token "eyJhbGci..." \
  -pkey "JduzsUuRvGVPRHvIYwLv"

# 阿里云 HLS 标准加密
./m3u8 -u "https://example.aliyundoc.com/test.m3u8?MediaId=xxx" \
  -mts-token "your-token"
```

Key Hook 优先级：**显式 `-decrypt-script` > 内置 Provider > `decrypt.yaml` / 自动发现脚本 > 原始 key**。指定 `-decrypt-script` 时，内置 Provider 的 Key 处理让位于脚本；URL 预处理（`-drm-token` / `-mts-token`）仍会执行。

调试参考脚本：`scripts/tencent-vod-simpleaes.star`、`scripts/aliyun-hls-standard.star`。详见 [scripts/README.md](scripts/README.md)。

**能力边界：** 支持标准 HLS AES-128、腾讯云 SimpleAES、阿里云 HLS 标准加密。不支持阿里云私有加密 / License 加密（SDK 专有），也不支持 FairPlay / Widevine 等商业 DRM。

## 解密脚本

对于非标准加密（自定义 key 派生、SAMPLE-AES 等），可将脚本放在 `scripts/` 目录，或在 `decrypt.yaml` 中配置匹配规则（参考 `decrypt.yaml.example`）。

脚本选择优先级：**CLI `-decrypt-script` → `decrypt.yaml` 规则 → 按 METHOD / 域名自动发现**。

| 参数 | 说明 |
|------|------|
| `-decrypt-script` | 显式指定脚本路径（优先级最高） |
| `-decrypt-config` | 按 host/method 匹配的配置文件 |
| `-scripts-dir` | 自动发现脚本的脚本库目录 |

```bash
# 使用配置文件
./m3u8 -u=https://example.com/index.m3u8 -decrypt-config decrypt.yaml

# 自动匹配 scripts/AES-128.star 或 scripts/example.com.py
./m3u8 -u=https://example.com/index.m3u8 -scripts-dir scripts
```

`.star` 脚本在沙箱中运行；其他扩展名（如 `.py`）通过长驻外部进程以 JSON stdin/stdout 通信。未匹配到脚本且 `METHOD` 为 `AES-128` 时，行为与无脚本时完全一致。

详细 Hook API、JSON 协议与示例见 [scripts/README.md](scripts/README.md)。

## HTTP API

通过 `m3u8 serve` 启动 HTTP API 服务，支持解析 M3U8、创建异步下载任务、查询进度与下载成品文件。

服务启动时会自动加载工作目录下的 `decrypt.yaml`（脚本匹配规则与 CLI 一致）。重启后 pending / running 状态的任务会自动恢复，并从任务目录中已有分片继续下载。

> 云点播凭证（`-drm-token`、`-pkey`、`-mts-token`）**仅支持 CLI**，HTTP API 暂不支持。下载腾讯云 / 阿里云加密资源请使用 CLI 模式。

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

### 服务选项

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

### 任务状态

`pending` → `running` → `completed` | `failed` | `cancelled` | `expired`

### 使用示例

解析 M3U8（默认返回前 5 个分片，加 `?full=true` 返回全部）：

```bash
curl -X POST http://localhost:8080/api/v1/parse \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/index.m3u8","proxy":"http://127.0.0.1:7890"}'
```

创建下载任务并轮询状态：

```bash
# 创建任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com/index.m3u8","filename":"myvideo","concurrency":25,"to_mp4":true,"proxy":"http://127.0.0.1:7890"}'

# 查询任务（将 <taskID> 替换为返回的 task_id）
curl http://localhost:8080/api/v1/tasks/<taskID>

# 下载成品（任务 status 为 completed 后）
curl -OJ http://localhost:8080/api/v1/tasks/<taskID>/download

# 取消进行中的任务
curl -X DELETE http://localhost:8080/api/v1/tasks/<taskID>
```

启用认证时，在上述请求中增加 `-H "X-API-Key: your-secret-key"`。

### 解析请求体

| 字段 | 类型 | 说明 |
|------|------|------|
| `url` | string | M3U8 地址（**必填**） |
| `proxy` | string | HTTP 代理地址 |

### 创建任务请求体

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `url` | string | | M3U8 地址（**必填**） |
| `filename` | string | `main` | 输出文件名 |
| `concurrency` | int | `25` | 下载并发数 |
| `to_mp4` | bool | `true` | 合并后转 MP4 |
| `proxy` | string | | HTTP 代理地址 |

## 开发

项目使用 Go Modules + vendor 模式，构建时从 `vendor/` 读取依赖，保证可复现。

```bash
make vendor   # 安装依赖到 vendor/
make build    # 编译
make test     # 运行测试
```

交叉编译：

```bash
make build-linux
make build-darwin-arm64
make build-windows
```

## 下载

[二进制文件](https://github.com/0377/m3u8/releases)

[上游发布](https://github.com/oopsguy/m3u8/releases)

## 参考资料

- [TS科普 2 包头](https://blog.csdn.net/cabbage2008/article/details/49281729)
- [HTTP Live Streaming draft-pantos-http-live-streaming-23](https://tools.ietf.org/html/draft-pantos-http-live-streaming-23#section-4.3.4.2)
- [MPEG transport stream - Wikipedia](https://en.wikipedia.org/wiki/MPEG_transport_stream)

## License

[MIT License](./LICENSE)
