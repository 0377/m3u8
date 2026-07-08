package crypt

// ProviderIntegration wires cloud VOD providers without importing crypt/provider
// (avoids import cycle: crypt <-> provider).
type ProviderIntegration struct {
	PreprocessURL    func(rawURL string, params ProviderParams) string
	DetectFromURL      func(rawURL string) string
	ValidateParams     func(id string, params ProviderParams) error
	DetectFromKeyURI   func(keyURI string) string
	NewDecryptor       func(id string) (Decryptor, error)
}

var providerIntegration ProviderIntegration

func RegisterProviderIntegration(pi ProviderIntegration) {
	providerIntegration = pi
}
