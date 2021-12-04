package secret

import (
	"fmt"
	"strings"
)

type path struct {
	namespace string
	secret    string
	key       string
	isDir     bool
}

// splitPath returns a path
func splitPath(name string) (*path, error) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) < 2 || len(parts) > 3 {
		return nil, fmt.Errorf("invalid path: %s", name)
	}

	p := &path{
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

func (p path) Namespace() string {
	return p.namespace
}

func (p path) Secret() string {
	return p.secret
}

func (p path) Key() string {
	return p.key
}

func (p path) IsDir() bool {
	return p.isDir
}
