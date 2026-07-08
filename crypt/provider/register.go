package provider

import (
	"fmt"

	"github.com/0377/m3u8/crypt"
)

func init() {
	crypt.RegisterProviderIntegration(crypt.ProviderIntegration{
		PreprocessURL: func(rawURL string, params crypt.ProviderParams) string {
			return PreprocessURL(rawURL, ProviderParams{
				DrmToken: params.DrmToken,
				Pkey:     params.Pkey,
				MtsToken: params.MtsToken,
			})
		},
		DetectFromURL: DetectFromURL,
		ValidateParams: func(id string, params crypt.ProviderParams) error {
			return ValidateParams(id, ProviderParams{
				DrmToken: params.DrmToken,
				Pkey:     params.Pkey,
				MtsToken: params.MtsToken,
			})
		},
		DetectFromKeyURI: DetectFromKeyURI,
		NewDecryptor:     newProviderDecryptor,
	})
}

func newProviderDecryptor(id string) (crypt.Decryptor, error) {
	switch id {
	case IDTencentSimpleAES:
		return NewTencentDecryptor(), nil
	case IDAliyunHLSStandard:
		return NewAliyunDecryptor(), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", id)
	}
}
