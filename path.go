package secretfs

import (
	"fmt"
	"strings"
)

// PathElement denotes the elements of a path
// NAMESPACE/SECRET[/KEY]
type PathElement int8

const (
	NAMESPACE PathElement = iota
	SECRET
	KEY
)

type Path map[PathElement]string

// NewPath returns Path
func NewPath(name string) (Path, error) {
	p := Path{}

	for i, v := range strings.Split(strings.Trim(name, "/"), "/") {
		p[PathElement(i)] = v
	}

	if len(p) < 2 || len(p) > 3 {
		return nil, fmt.Errorf("invalid path: %s", name)
	}

	return p, nil
}

func (p Path) IsDir() bool {
	return len(p) == 2
}
