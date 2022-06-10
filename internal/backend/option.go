package backend

import (
	"time"

	"go.uber.org/zap"
)

// Option represents a functional Option
type Option func(*backend)

/*
// WithSecretType configures a custom secret type
// TODO: does this belong here?
func WithSecretType(t corev1.SecretType) Option {
	return func(b *backend) {
		b.secretType = t
	}
}
*/

// WithSecretPrefix configures a custom secret prefix
func WithSecretPrefix(x string) Option {
	return func(b *backend) {
		b.prefix = x
	}
}

// WithSecretSuffix configures a custom secret prefix
func WithSecretSuffix(x string) Option {
	return func(b *backend) {
		b.suffix = x
	}
}

// WithTimeout configures a custom request timeout
func WithTimeout(t time.Duration) Option {
	return func(b *backend) {
		b.timeout = t
	}
}

// WithSecretLabels configures a custom secret labels
func WithSecretLabels(labels map[string]string) Option {
	return func(b *backend) {
		b.labels = labels
	}
}

// WithLogger configures a logger
func WithLogger(l *zap.SugaredLogger) Option {
	return func(b *backend) {
		b.l = l
	}
}
