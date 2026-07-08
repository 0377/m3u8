package provider

import (
	"net/url"
	"strings"
)

func InsertDRMToken(rawURL, token string) string {
	if rawURL == "" || token == "" || strings.Contains(rawURL, "voddrm.token.") {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	i := strings.LastIndex(u.Path, "/")
	if i < 0 {
		return rawURL
	}
	u.Path = u.Path[:i+1] + "voddrm.token." + token + "." + u.Path[i+1:]
	return u.String()
}

func AppendMtsToken(rawURL, token string) string {
	if rawURL == "" || token == "" {
		return rawURL
	}
	if strings.Contains(rawURL, "MtsHlsUriToken=") {
		return rawURL
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + "MtsHlsUriToken=" + url.QueryEscape(token)
}

// PreprocessURL applies provider-specific URL transforms before M3U8 fetch.
func PreprocessURL(rawURL string, params ProviderParams) string {
	out := rawURL
	if params.DrmToken != "" {
		out = InsertDRMToken(out, params.DrmToken)
	}
	if params.MtsToken != "" {
		out = AppendMtsToken(out, params.MtsToken)
	}
	return out
}
