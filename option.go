package sekretsfs

import (
	"time"

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

// WithSecretLabels configures a custom secret labels
func WithSecretLabels(labels map[string]string) Option {
	return func(s *sekretsFs) {
		s.labels = labels
	}
}

// WithTimeout configures a custom request timeout
func WithTimeout(t time.Duration) Option {
	return func(s *sekretsFs) {
		s.timeout = t
	}
}

// WithLogger configures a logger
func WithLogger(l *zap.SugaredLogger) Option {
	return func(s *sekretsFs) {
		s.l = l
	}
}
