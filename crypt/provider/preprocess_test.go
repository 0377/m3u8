package provider

import "testing"

func TestInsertDRMToken(t *testing.T) {
	in := "https://1500014561.vod2.myqcloud.com/a/b/adp.12.m3u8?t=1&sign=2"
	token := "abc123"
	want := "https://1500014561.vod2.myqcloud.com/a/b/voddrm.token.abc123.adp.12.m3u8?t=1&sign=2"
	got := InsertDRMToken(in, token)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestInsertDRMToken_already_present(t *testing.T) {
	in := "https://x.vod2.myqcloud.com/a/voddrm.token.tok.test.m3u8"
	got := InsertDRMToken(in, "new")
	if got != in {
		t.Fatalf("should not modify: %q", got)
	}
}

func TestAppendMtsToken(t *testing.T) {
	in := "https://example.com/test.m3u8?MediaId=abc"
	got := AppendMtsToken(in, "tok99")
	if got != "https://example.com/test.m3u8?MediaId=abc&MtsHlsUriToken=tok99" {
		t.Fatalf("got %q", got)
	}
}

func TestAppendMtsToken_already_present(t *testing.T) {
	in := "https://example.com/test.m3u8?MtsHlsUriToken=old"
	got := AppendMtsToken(in, "new")
	if got != in {
		t.Fatalf("should not modify")
	}
}
