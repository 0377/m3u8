# 解密脚本库

本目录存放可插拔解密脚本。工具按 **CLI 指定 → `decrypt.yaml` 规则 → 自动发现** 的优先级选择脚本；未匹配到脚本且 `METHOD` 为 `AES-128` 时，行为与无脚本时完全一致。

## 脚本命名与自动发现

将脚本放在 `scripts_dir`（默认 `scripts/`）下，工具会按以下顺序尝试匹配：

| 优先级 | 文件名模式 | 示例 |
|--------|-----------|------|
| 1 | `<METHOD>.star` / `<METHOD>.py` | `AES-128.star`、`SAMPLE-AES.py` |
| 2 | `<hostname>.star` / `<hostname>.py` | `example.com.star` |

说明：

- `<METHOD>` 与 M3U8 中 `#EXT-X-KEY:METHOD=...` 的值一致（区分大小写）。
- `<hostname>` 为 M3U8 URL 的域名（不含端口）。
- 扩展名决定运行时：`.star` 走嵌入式 Starlark，其他扩展名（如 `.py`、`.sh`）走外部长驻进程。
- 带后缀的示例名（如 `AES-128-custom.star`）**不会**被 `AES-128` 自动匹配；请通过 `-decrypt-script` 或 `decrypt.yaml` 显式指定。

## 三个 Hook 与回退行为

脚本可实现以下三个可选函数（Starlark）或对应 JSON `hook`（外部进程）。未实现的 Hook 自动回退下一层。

### 1. Key Hook — `decrypt_key` / `hook: "key"`

在解析 M3U8、拉取 key URI 原始响应后调用，用于 key/IV 二次派生。

**回退：** 未实现时，直接使用 M3U8 中的原始 key 与 IV。

### 2. Segment Hook — `decrypt_segment` / `hook: "segment"`

每个 TS 分片下载后调用，用于自定义分片解密。

**回退链（分片路径）：**

1. 若实现 `decrypt_full` → 调用 Full Hook，跳过内置解密
2. 否则若实现 `decrypt_segment` → 调用 Segment Hook
3. 否则若 `METHOD == AES-128` → 内置 `AES-128-CBC` 解密
4. 否则报错，提示添加脚本

### 3. Full Hook — `decrypt_full` / `hook: "full"`

完全接管分片解密，忽略内置 AES-128 与 Segment Hook。

**回退：** 未实现时，继续尝试 Segment Hook 或内置解密。

## Starlark API 参考

`.star` 脚本使用 [Starlark](https://github.com/google/starlark-go) 语法。仅实现需要的函数即可。

### `decrypt_key(raw_key, method, uri, iv, m3u8_url)`

| 参数 | 类型 | 说明 |
|------|------|------|
| `raw_key` | bytes | key URI 返回的原始字节 |
| `method` | string | `#EXT-X-KEY` 的 METHOD |
| `uri` | string | key URI |
| `iv` | string | M3U8 中的 IV 字符串（可能为空） |
| `m3u8_url` | string | 当前播放列表 URL |

**返回值：**

- `{"key": <bytes>, "iv": <bytes或string>}` — 推荐
- 或直接返回 `bytes` 作为 key（IV 沿用 M3U8 中的值）

### `decrypt_segment(ciphertext, key, iv, index, uri)`

| 参数 | 类型 | 说明 |
|------|------|------|
| `ciphertext` | bytes | 分片密文 |
| `key` | bytes | 经 Key Hook 处理后的 key |
| `iv` | bytes | 经 Key Hook 处理后的 IV |
| `index` | int | 分片序号（从 0 起） |
| `uri` | string | 分片 URL |

**返回值：** `bytes` 明文

### `decrypt_full(ciphertext, index, uri, method, key, iv)`

| 参数 | 类型 | 说明 |
|------|------|------|
| `ciphertext` | bytes | 分片密文 |
| `index` | int | 分片序号 |
| `uri` | string | 分片 URL |
| `method` | string | 加密 METHOD |
| `key` | bytes | 当前 key（可能为空） |
| `iv` | bytes | 当前 IV（可能为空） |

**返回值：** `bytes` 明文

### 内置辅助函数

| 函数 | 说明 |
|------|------|
| `aes128_cbc_decrypt(ciphertext, key, iv)` | 标准 AES-128-CBC 解密（含 PKCS7 去填充），返回 `bytes` |
| `sha256(data)` | SHA-256 摘要，参数为 `bytes`，返回 32 字节 `bytes` |
| `hex_decode(s)` | 十六进制字符串 → `bytes` |
| `base64_decode(s)` | 标准 Base64 字符串 → `bytes` |
| `aes_cbc_decrypt_zero_iv(ciphertext, key)` | AES-CBC 解密，IV 为全零；`key` 可为 16/24/32 字节（腾讯云 SimpleAES 使用 32 字节 SHA256 摘要） |

示例（Segment Hook 调用内置 AES）：

```python
def decrypt_segment(ciphertext, key, iv, index, uri):
    return aes128_cbc_decrypt(ciphertext, key, iv)
```

完整示例见本目录下的 `AES-128-custom.star`；云点播参考脚本见下文。

## 云点播内置 Provider

工具内置腾讯云 SimpleAES 与阿里云 HLS 标准加密的 **URL 预处理** 与 **Key 二次解密**。检测到对应云点播特征时自动启用，**通常无需** `-decrypt-script`。

### 自动检测规则

| Provider | 触发条件（满足其一） | Key 处理 |
|----------|---------------------|----------|
| 腾讯云 SimpleAES | M3U8 域名为 `*.vod2.myqcloud.com`；或 key URI 含 `drm.vod2.myqcloud.com` 且 `drmType=SimpleAES` | `SHA256(pkey)` → AES-CBC（零 IV）解密密文 key |
| 阿里云 HLS 标准加密 | M3U8 URL 含 `MtsHlsUriToken=`；或 key URI 含 `Ciphertext=` | 响应为 16 字节二进制或 Base64 文本 → 16 字节 AES key |

### CLI 凭证参数

凭证通过 CLI 手动传入，不写入 `decrypt.yaml`：

| 参数 | 用途 |
|------|------|
| `-drm-token` | 腾讯云 DrmToken；用于在 M3U8 路径中插入 `voddrm.token.{token}` |
| `-pkey` | 腾讯云 SimpleAES 播放密钥 |
| `-mts-token` | 阿里云 `MtsHlsUriToken`，拼接到 M3U8 URL 查询参数 |

```bash
# 腾讯云 SimpleAES（推荐：内置 Provider，无需脚本）
m3u8 -u "https://1500014561.vod2.myqcloud.com/.../index.m3u8" \
  -drm-token "eyJhbGci..." \
  -pkey "your-pkey"

# 阿里云 HLS 标准加密
m3u8 -u "https://example.aliyundoc.com/test.m3u8?MediaId=xxx" \
  -mts-token "your-token"
```

### 编排优先级

Key Hook 路径：**显式 `-decrypt-script` > 内置 Provider > `decrypt.yaml` / 自动发现脚本 > 原始 key**。

指定 `-decrypt-script` 时，内置 Provider 的 Key 处理让位于脚本；URL 预处理（`-drm-token` / `-mts-token`）仍会执行。

### 参考脚本（调试用）

本目录提供与内置 Provider 等价的 Starlark 参考实现，便于对照算法或二次开发：

| 脚本 | 说明 |
|------|------|
| `tencent-vod-simpleaes.star` | 演示 `sha256` + `aes_cbc_decrypt_zero_iv`；内含文档示例 `PKEY`，**勿用于生产** |
| `aliyun-hls-standard.star` | 演示 `base64_decode` 路径；16 字节二进制响应直接透传 |

生产环境请使用内置 Provider 配合 `-pkey` / `-drm-token` / `-mts-token`，不要将真实密钥写入脚本或提交到版本库。

## 外部脚本 JSON 协议

非 `.star` 脚本以**长驻子进程**运行：stdin 写入一行 JSON 请求，stdout 读取一行 JSON 响应。进程在首次调用时启动，下载结束后关闭。超时由 `decrypt.yaml` 的 `external_timeout` 控制（默认 30s）。

> **性能提示：** Segment Hook 对每个分片都会跨进程通信，开销较大。高频场景优先使用 Starlark（`.star`）；外部进程更适合 Key Hook、Full Hook 或低频自定义 METHOD。

### 请求格式

**Key Hook：**

```json
{"id": 1, "hook": "key", "method": "AES-128", "raw_key": "<base64>", "iv": "...", "m3u8_url": "..."}
```

**Segment Hook：**

```json
{"id": 2, "hook": "segment", "ciphertext": "<base64>", "key": "<base64>", "iv": "...", "index": 0, "uri": "..."}
```

**Full Hook：**

```json
{"id": 3, "hook": "full", "ciphertext": "<base64>", "index": 0, "uri": "...", "method": "CUSTOM"}
```

字段说明：

- `id`：单调递增的请求 ID，响应必须原样回传
- `raw_key` / `ciphertext` / `key` / `data`：标准 Base64 编码
- `iv`：字符串形式（Segment 请求中为处理后的 IV）

### 响应格式

**成功：**

```json
{"id": 1, "ok": true, "key": "<base64>", "iv": "..."}
{"id": 2, "ok": true, "data": "<base64>"}
```

**失败：**

```json
{"id": 2, "ok": false, "error": "decryption failed: invalid padding"}
```

**未实现某 Hook：** 返回 `{"id": N, "ok": false, "error": "not implemented"}`，工具将回退到下一层（与 Starlark 未定义函数行为一致）。

### Python 最小示例

```python
#!/usr/bin/env python3
import sys, json, base64

for line in sys.stdin:
    req = json.loads(line)
    rid = req["id"]
    hook = req["hook"]
    if hook == "key":
        resp = {"id": rid, "ok": True, "key": req["raw_key"], "iv": req.get("iv", "")}
    elif hook == "segment":
        data = base64.b64decode(req["ciphertext"])
        resp = {"id": rid, "ok": True, "data": base64.b64encode(data).decode()}
    else:
        resp = {"id": rid, "ok": False, "error": "not implemented"}
    print(json.dumps(resp), flush=True)
```

参考实现：`crypt/testdata/echo_decrypt.py`。

## 配置与 CLI

```bash
# 显式指定脚本（优先级最高）
m3u8 -u https://example.com/index.m3u8 -decrypt-script scripts/AES-128-custom.star

# 使用配置文件（复制 decrypt.yaml.example 为 decrypt.yaml）
m3u8 -u https://example.com/index.m3u8 -decrypt-config decrypt.yaml

# 指定脚本库目录
m3u8 -u https://example.com/index.m3u8 -scripts-dir scripts
```

`decrypt.yaml` 支持按 `host`、`method`、`url` 子串匹配规则，详见项目根目录的 `decrypt.yaml.example`。

## 安全提示

- **仅运行可信脚本。** 外部进程脚本拥有与运行用户相同的系统权限，可执行任意代码。
- 脚本路径必须为 `-decrypt-script` 显式指定，或位于 `scripts_dir` 目录内；工具不会从 `PATH` 搜索脚本。
- Starlark 脚本在沙箱中执行，禁止直接的文件 IO 与网络访问，仅能通过预置 binding（如 `aes128_cbc_decrypt`）处理数据。
- 请勿将含密钥派生逻辑的脚本提交到公开仓库；敏感逻辑应放在本地 `decrypt.yaml` 规则指向的私有脚本中。
- 从不可信来源获取的 M3U8 / key URI 可能触发恶意脚本逻辑，请谨慎处理。
