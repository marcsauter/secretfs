package secretfs

import (
	"fmt"
	"strings"
)

// secretPathElement denotes the elements of a path
// NAMESPACE/SECRET[/KEY]
type secretPathElement int8

// PathElements
const (
	NAMESPACE secretPathElement = iota
	SECRET
	KEY
)

type secretPath map[secretPathElement]string

// newSecretPath returns Path
func newSecretPath(name string) (secretPath, error) {
	p := secretPath{}

	for i, v := range strings.Split(strings.Trim(name, "/"), "/") {
		p[secretPathElement(i)] = v
	}

	if len(p) < 2 || len(p) > 3 {
		return nil, fmt.Errorf("invalid path: %s", name)
	}

	return p, nil
}

func (p secretPath) isDir() bool {
	return len(p) == 2
}
