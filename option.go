package sekretsfs

import (
	"go.uber.org/zap"
)

// Option represents a functional Option
type Option func(*sekretsFs)

// WithSecretPrefix configures a custom secret prefix
func WithSecretPrefix(x string) Option {
	return func(s *sekretsFs) {
		s.prefix = x
	}
}

// WithSecretSuffix configures a custom secret suffix
func WithSecretSuffix(x string) Option {
	return func(s *sekretsFs) {
		s.suffix = x
	}
}

// WithLogger configures a logger
func WithLogger(l *zap.SugaredLogger) Option {
	return func(s *sekretsFs) {
		s.l = l
	}
}
