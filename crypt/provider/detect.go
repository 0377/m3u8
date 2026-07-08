package provider

import (
	"fmt"
	"net/url"
	"strings"
)

type ProviderParams struct {
	DrmToken string
	Pkey     string
	MtsToken string
}

func DetectFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if strings.HasSuffix(host, ".vod2.myqcloud.com") {
		return IDTencentSimpleAES
	}
	if strings.Contains(rawURL, "MtsHlsUriToken=") {
		return IDAliyunHLSStandard
	}
	return ""
}

func DetectFromKeyURI(keyURI string) string {
	if strings.Contains(keyURI, "drm.vod2.myqcloud.com") && strings.Contains(keyURI, "drmType=SimpleAES") {
		return IDTencentSimpleAES
	}
	if strings.Contains(keyURI, "Ciphertext=") {
		return IDAliyunHLSStandard
	}
	return ""
}

func ValidateParams(id string, params ProviderParams) error {
	if id != IDTencentSimpleAES {
		return nil
	}
	var missing []string
	if params.DrmToken == "" {
		missing = append(missing, "-drm-token")
	}
	if params.Pkey == "" {
		missing = append(missing, "-pkey")
	}
	if len(missing) > 0 {
		return fmt.Errorf("腾讯云 SimpleAES 需要参数: %s", strings.Join(missing, ", "))
	}
	return nil
}
