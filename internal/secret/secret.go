// Package secret provides a independent secret
package secret

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/marcsauter/sekretsfs/internal/io"
	"github.com/spf13/afero"
)

// Secret is the corev1.Secret without k8s specific data
type Secret struct {
	name      string // absolute name namespace/secret[/key]
	namespace string
	secret    string
	key       string
	data      map[string][]byte
	isDir     bool
	size      int64
	mtime     time.Time
	mode      fs.FileMode

	TLS bool // TODO: corev1.SecretTypeTLS
}

// New returns a new Secret
// Secret is also os.FileInfo
func New(name string) (*Secret, error) {
	p, err := splitPath(name)
	if err != nil {
		return nil, err
	}

	s := &Secret{
		name:      name,
		namespace: p.Namespace(),
		secret:    p.Secret(),
		key:       p.Key(),
	}

	s.isDir = false
	s.mode = fs.FileMode(0)
	s.mtime = time.Now()

	if p.IsDir() {
		s.isDir = true
		s.mode = fs.ModeDir
	}

	return s, nil
}

var _ io.Sekreter = (*Secret)(nil)

// Namespace returns the namespace name
func (s *Secret) Namespace() string {
	return s.namespace
}

// Path returns the secret name
func (s *Secret) Path() string {
	return s.secret
}

// SetData sets the secret data map
func (s *Secret) SetData(data map[string][]byte) {
	s.data = data
	s.size = int64(len(s.data))
}

// Data returns the secret data map
func (s *Secret) Data() map[string][]byte {
	return s.data
}

// Get a secret entry
func (s *Secret) Get(key string) ([]byte, bool) {
	v, ok := s.data[key]

	return v, ok
}

// Add a key/value pair, do not overwrite an existing.
func (s *Secret) Add(key string, value []byte) error {
	if s.data == nil {
		s.data = make(map[string][]byte)
	}

	if _, ok := s.data[key]; ok {
		return afero.ErrFileExists
	}

	s.data[key] = value
	s.size = int64(len(s.data))
	s.mtime = time.Now()

	return nil
}

// Update a secret key/value pair, overwrite an existing.
func (s *Secret) Update(key string, value []byte) {
	if s.data == nil {
		s.data = make(map[string][]byte)
	}

	s.data[key] = value
	s.size = int64(len(s.data))
	s.mtime = time.Now()
}

// Delete a key/value pair
func (s *Secret) Delete(key string) error {
	if _, ok := s.data[key]; !ok {
		return afero.ErrFileNotFound
	}

	delete(s.data, key)
	s.size = int64(len(s.data))
	s.mtime = time.Now()

	return nil
}

var _ os.FileInfo = (*Secret)(nil)

// Name returns the base name
func (s *Secret) Name() string {
	return filepath.Base(s.name)
}

// Size returns length in bytes for keys
func (s *Secret) Size() int64 {
	return s.size
}

// Mode returns file mode bits
func (s *Secret) Mode() fs.FileMode {
	return s.mode
}

// ModTime returns file modification time
func (s *Secret) ModTime() time.Time {
	return s.mtime
}

// IsDir returns true for a secret, false for a key
func (s *Secret) IsDir() bool {
	return s.isDir
}

// Sys returns underlying data source (can return nil)
func (s *Secret) Sys() interface{} {
	return s
}
