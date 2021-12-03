package secretfs

import (
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

// Option represents a functional Option
type Option func(*secretFs)

// WithSecretType configures a custom secret type
// TODO: does this belong here?
func WithSecretType(t corev1.SecretType) Option {
	return func(s *secretFs) {
		s.secretType = t
	}
}

// WithSecretPrefix configures a custom secret prefix
// TODO: does this belong here?
func WithSecretPrefix(x string) Option {
	return func(s *secretFs) {
		s.prefix = x
	}
}

// WithSecretSuffix configures a custom secret prefix
// TODO: does this belong here?
func WithSecretSuffix(x string) Option {
	return func(s *secretFs) {
		s.suffix = x
	}
}

// WithTimeout configures a custom request timeout
func WithTimeout(t time.Duration) Option {
	return func(s *secretFs) {
		s.timeout = t
	}
}

// WithLogger configures a logger
func WithLogger(l *zap.SugaredLogger) Option {
	return func(s *secretFs) {
		s.l = l
	}
}
