def decrypt_key(raw_key, method, uri, iv, m3u8_url):
    return {"key": raw_key[::-1], "iv": iv}

def decrypt_segment(ciphertext, key, iv, index, uri):
    return aes128_cbc_decrypt(ciphertext, key, iv)
