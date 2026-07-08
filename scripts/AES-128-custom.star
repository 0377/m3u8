# 示例：Key Hook 透传 — 原样返回 key/IV，分片解密仍走内置 AES-128。
# 适用于仅需观察或占位 Key Hook 的场景；实际派生逻辑请按需修改。
#
# 使用方式（任选其一）：
#   m3u8 -u <URL> -decrypt-script scripts/AES-128-custom.star
#   或在 decrypt.yaml 中按 method/host 规则指向本脚本

def decrypt_key(raw_key, method, uri, iv, m3u8_url):
    return {"key": raw_key, "iv": iv}
