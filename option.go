package secfs

import (
	"time"
)

// Option represents a functional Option
type Option func(*secfs)

// WithSecretPrefix configures a custom secret prefix
func WithSecretPrefix(x string) Option {
	return func(s *secfs) {
		s.prefix = x
	}
}

// WithSecretSuffix configures a custom secret suffix
func WithSecretSuffix(x string) Option {
	return func(s *secfs) {
		s.suffix = x
	}
}

// WithSecretLabels configures a custom secret labels
func WithSecretLabels(labels map[string]string) Option {
	return func(s *secfs) {
		s.labels = labels
	}
}

// WithTimeout configures a custom request timeout
func WithTimeout(t time.Duration) Option {
	return func(s *secfs) {
		s.timeout = t
	}
}
