package sekretsfs

import (
	"go.uber.org/zap"
)

// Option represents a functional Option
type Option func(*sekretsFs)

// WithSecretPrefix configures a custom secret prefix
// TODO: does this belong here?
func WithSecretPrefix(x string) Option {
	return func(s *sekretsFs) {
		s.prefix = x
	}
}

// WithSecretSuffix configures a custom secret prefix
// TODO: does this belong here?
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
