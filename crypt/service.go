package crypt

import (
	"errors"
	"fmt"
)

type ServiceProviderOptions struct {
	ActiveID string
	Params   ProviderParams
}

type Service struct {
	registry *Registry
	builtin  *BuiltinDecryptor
	prov     ServiceProviderOptions
}

func NewService(registry *Registry, prov ServiceProviderOptions) *Service {
	return &Service{registry: registry, builtin: &BuiltinDecryptor{}, prov: prov}
}

func (s *Service) Close() error {
	if s.registry != nil {
		return s.registry.Close()
	}
	return nil
}

func (s *Service) SetActiveProvider(id string) {
	s.prov.ActiveID = id
}

func (s *Service) DetectProviderFromKeyURIs(uris []string) string {
	if providerIntegration.DetectFromKeyURI == nil {
		return ""
	}
	for _, uri := range uris {
		if id := providerIntegration.DetectFromKeyURI(uri); id != "" {
			return id
		}
	}
	return ""
}

func (s *Service) providerDecryptor() (Decryptor, error) {
	if providerIntegration.NewDecryptor == nil {
		return nil, fmt.Errorf("unknown provider %q", s.prov.ActiveID)
	}
	return providerIntegration.NewDecryptor(s.prov.ActiveID)
}

func (s *Service) ProcessKey(ctx *Context, rawKey []byte, meta *KeyMeta) (KeyMaterial, error) {
	ctx.Provider = s.prov.ActiveID
	ctx.Params = s.prov.Params

	var d Decryptor
	var err error
	if !s.registry.HasExplicitScript(ctx) && s.prov.ActiveID != "" {
		d, err = s.providerDecryptor()
	} else {
		d, err = s.registry.Resolve(ctx)
	}
	if err != nil {
		return KeyMaterial{}, err
	}
	key, iv, err := d.ProcessKey(ctx, rawKey, meta)
	if err != nil {
		return KeyMaterial{}, err
	}
	return KeyMaterial{Key: key, IV: iv}, nil
}

func (s *Service) DecryptSegment(ctx *Context, ciphertext, key, iv []byte) ([]byte, error) {
	ctx.Key = key
	ctx.IV = iv
	d, err := s.registry.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	if plaintext, ok, err := d.DecryptFull(ctx, ciphertext); ok {
		return plaintext, err
	}
	plaintext, err := d.DecryptSegment(ctx, ciphertext, key, iv)
	if err == nil {
		return plaintext, nil
	}
	if !errors.Is(err, ErrNotImplemented) {
		return nil, err
	}
	if ctx.Method == "AES-128" {
		return s.builtin.DecryptSegment(ctx, ciphertext, key, iv)
	}
	return nil, err
}
