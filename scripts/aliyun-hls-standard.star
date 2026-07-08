# 参考实现：与内置 aliyun-hls-standard Provider 的 Key 处理等价。
#
# ⚠️ 仅用于调试与二次开发对照，生产环境请使用内置 Provider（无需 -decrypt-script）：
#   m3u8 -u <URL> -mts-token <token>
#
# 算法：key URI 响应若为 16 字节二进制则直接使用，否则按 Base64 文本解码为 16 字节 AES key。

def _bytes_to_str(data):
    s = ""
    for b in data:
        s += chr(b)
    return s

def decrypt_key(raw_key, method, uri, iv, m3u8_url):
    if len(raw_key) == 16:
        return {"key": raw_key, "iv": iv}
    key = base64_decode(_bytes_to_str(raw_key).strip())
    return {"key": key, "iv": iv}
