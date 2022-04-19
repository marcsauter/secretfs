// Package item represent an entry in the map of a Kubernetes secret
// item implements afero.File
package item

import (
	"os"

	"github.com/marcsauter/sekretsfs/internal/secret"
	"github.com/spf13/afero"
)

// Item is a k8s secret key/value entry implementing the afero.File interface
type Item struct {
	ref   *secret.Secret
	key   string
	value []byte
}

// New returns a new Item
func New(ref *secret.Secret, key string, value []byte) *Item {
	return &Item{
		ref:   ref,
		key:   key,
		value: value,
	}
}

var _ afero.File = (*Item)(nil)

// Close io.Closer
func (s *Item) Close() error {
	if err := s.Sync(); err != nil {
		return err
	}

	s.ref = nil

	return nil
}

// Read io.Reader
// https://pkg.go.dev/io#Reader
func (s *Item) Read(p []byte) (n int, err error) {
	return
}

// ReadAt io.ReaderAt
// https://pkg.go.dev/io#ReaderAt
func (s *Item) ReadAt(p []byte, off int64) (n int, err error) {
	return
}

// Seek io.Seeker
// https://pkg.go.dev/io#Seeker
func (s *Item) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

// Write io.Writer
// https://pkg.go.dev/io#Writer
func (s *Item) Write(p []byte) (n int, err error) {
	return
}

// WriteAt io.WriterAt
// https://pkg.go.dev/io#WriterAt
func (s *Item) WriteAt(p []byte, off int64) (n int, err error) {
	return
}

// Name returns the secret name
func (s *Item) Name() string {
	return s.key
}

// Readdir afero.File
func (s *Item) Readdir(count int) ([]os.FileInfo, error) {
	return []os.FileInfo{}, nil
}

// Readdirnames afero.File
func (s *Item) Readdirnames(n int) ([]string, error) {
	return []string{}, nil
}

// Stat afero.File
func (s *Item) Stat() (os.FileInfo, error) {
	return nil, nil
}

// Sync afero.File
func (s *Item) Sync() error {
	return nil
}

// Truncate afero.File
func (s *Item) Truncate(size int64) error {
	if int64(len(s.value)) <= size {
		return nil
	}

	s.value = append([]byte{}, s.value[:size]...)

	return nil
}

// WriteString afero.File
func (s *Item) WriteString(st string) (ret int, err error) {
	return
}

// save function
// lock
// read secret again
// set value
// save secret
// unlock
