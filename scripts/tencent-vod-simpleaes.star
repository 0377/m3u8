# 参考实现：与内置 tencent-simpleaes Provider 的 Key 派生算法等价。
#
# ⚠️ 仅用于调试与二次开发对照，生产环境请使用内置 Provider（无需 -decrypt-script）：
#   m3u8 -u <URL> -drm-token <token> -pkey <pkey>
#
# 若显式指定本脚本，URL 预处理仍依赖 CLI 的 -drm-token；Starlark 无法读取 -pkey，
# 因此下方 PKEY 仅作文档示例向量，请替换为实际播放密钥，勿将真实密钥提交到仓库。
#
# 算法：sym_key = SHA256(pkey)，content_key = AES-CBC-Decrypt(raw_key, sym_key, zero IV)

PKEY = b"JduzsUuRvGVPRHvIYwLv"  # 腾讯云官方文档示例 pkey

def decrypt_key(raw_key, method, uri, iv, m3u8_url):
    sym_key = sha256(PKEY)
    content_key = aes_cbc_decrypt_zero_iv(raw_key, sym_key)
    return {"key": content_key, "iv": iv}
