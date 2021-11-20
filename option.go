package secretfs

import (
	"time"

	"go.uber.org/zap"
)

// Option represents a functional Option
type Option func(*secretFs)

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
