package provider

import "testing"

func TestDetectFromURL_tencent(t *testing.T) {
	id := DetectFromURL("https://1500014561.vod2.myqcloud.com/a/test.m3u8")
	if id != IDTencentSimpleAES {
		t.Fatalf("got %q", id)
	}
}

func TestDetectFromKeyURI_aliyun(t *testing.T) {
	id := DetectFromKeyURI(`https://example.com/key?Ciphertext=abc&MediaId=1`)
	if id != IDAliyunHLSStandard {
		t.Fatalf("got %q", id)
	}
}

func TestValidateParams_tencent_missing(t *testing.T) {
	err := ValidateParams(IDTencentSimpleAES, ProviderParams{})
	if err == nil {
		t.Fatal("expected error")
	}
}
