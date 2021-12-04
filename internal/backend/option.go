package backend

import (
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

// Option represents a functional Option
type Option func(*Backend)

// WithSecretType configures a custom secret type
// TODO: does this belong here?
func WithSecretType(t corev1.SecretType) Option {
	return func(b *Backend) {
		b.secretType = t
	}
}

/*
// WithSecretPrefix configures a custom secret prefix
// TODO: does this belong here?
func WithSecretPrefix(x string) Option {
	return func(s *Backend) {
		s.prefix = x
	}
}

// WithSecretSuffix configures a custom secret prefix
// TODO: does this belong here?
func WithSecretSuffix(x string) Option {
	return func(s *Backend) {
		s.suffix = x
	}
}
*/

// WithTimeout configures a custom request timeout
func WithTimeout(t time.Duration) Option {
	return func(b *Backend) {
		b.timeout = t
	}
}

// WithLogger configures a logger
func WithLogger(l *zap.SugaredLogger) Option {
	return func(b *Backend) {
		b.l = l
	}
}
