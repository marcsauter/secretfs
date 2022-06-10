package secfs

import (
	"path"
	"strings"
	"syscall"

	"github.com/marcsauter/secfs/internal/backend"
)

type secretPath struct {
	namespace string
	secret    string
	key       string
	isDir     bool
}

// newSecretPath returns the secretPath for name
func newSecretPath(name string) (*secretPath, error) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, syscall.EINVAL
	}

	p := &secretPath{
		namespace: parts[0],
		secret:    parts[1],
	}

	switch len(parts) {
	case 2:
		p.isDir = true
	case 3:
		p.key = parts[2]
	}

	return p, nil
}

func (p secretPath) Name() string {
	if p.key != "" {
		return p.key
	}

	return p.secret
}

func (p secretPath) Absolute() string {
	return path.Join(p.namespace, p.secret, p.key)
}

func (p secretPath) IsDir() bool {
	return p.isDir
}

var _ backend.Metadata = secretPath{}

func (p secretPath) Namespace() string {
	return p.namespace
}

func (p secretPath) Secret() string {
	return p.secret
}

func (p secretPath) Key() string {
	return p.key
}
