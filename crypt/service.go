package crypt

import "errors"

type Service struct {
	registry *Registry
	builtin  *BuiltinDecryptor
}

func NewService(registry *Registry) *Service {
	return &Service{registry: registry, builtin: &BuiltinDecryptor{}}
}

func (s *Service) ProcessKey(ctx *Context, rawKey []byte, meta *KeyMeta) (KeyMaterial, error) {
	d, err := s.registry.Resolve(ctx)
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
