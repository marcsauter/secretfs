// Package item represent an entry in the map of a Kubernetes secret
// item implements afero.File
package item

import (
	"io/fs"
	"os"
	"time"

	"github.com/marcsauter/sekretsfs/internal/secret"
	"github.com/spf13/afero"
)

// Item is a k8s secret key/value entry implementing the afero.File interface
type Item struct {
	ref *secret.Secret

	key   string
	value []byte
}

// New returns a new Item
func New(ref *secret.Secret, key string, value []byte) afero.File {
	return &Item{
		ref:   ref,
		key:   key,
		value: value,
	}
}

var _ os.FileInfo = (*Item)(nil)

// Close io.Closer
func (i *Item) Close() error {
	if i == nil {
		return os.ErrInvalid
	}

	defer func() {
		i.ref = nil
	}()

	return i.Sync()
}

// Read io.Reader
// https://pkg.go.dev/io#Reader
func (i *Item) Read(p []byte) (n int, err error) {
	return
}

// ReadAt io.ReaderAt
// https://pkg.go.dev/io#ReaderAt
func (i *Item) ReadAt(p []byte, off int64) (n int, err error) {
	return
}

// Seek io.Seeker
// https://pkg.go.dev/io#Seeker
func (i *Item) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

// Write io.Writer
// https://pkg.go.dev/io#Writer
func (i *Item) Write(p []byte) (n int, err error) {
	return
}

// WriteAt io.WriterAt
// https://pkg.go.dev/io#WriterAt
func (i *Item) WriteAt(p []byte, off int64) (n int, err error) {
	return
}

// Name returns the secret name (afero.File, io.FileInfo)
func (i *Item) Name() string {
	return i.key
}

// Readdir afero.File
func (i *Item) Readdir(count int) ([]os.FileInfo, error) {
	return []os.FileInfo{}, nil
}

// Readdirnames afero.File
func (i *Item) Readdirnames(n int) ([]string, error) {
	return []string{}, nil
}

// Stat afero.File
func (i *Item) Stat() (os.FileInfo, error) {
	return nil, nil
}

// Sync afero.File
func (i *Item) Sync() error {
	return nil
}

// Truncate afero.File
func (i *Item) Truncate(size int64) error {
	if int64(len(i.value)) <= size {
		return nil
	}

	i.value = append([]byte{}, i.value[:size]...)

	return nil
}

// WriteString afero.File
func (i *Item) WriteString(st string) (ret int, err error) {
	return
}

// Size returns the size in bytes (io.FileInfo)
func (i *Item) Size() int64 {
	return int64(len(i.value))
}

// Mode returns file mode bits
func (i *Item) Mode() fs.FileMode {
	return 0o666 // TODO:
}

// ModTime returns file modification time
func (i *Item) ModTime() time.Time {
	return time.Now() // TODO:
}

// IsDir always false for an Item
func (i *Item) IsDir() bool {
	return false
}

// Sys returns underlying data source (can return nil)
func (i *Item) Sys() interface{} {
	return nil // TODO:
}

// save function
// lock
// read secret again
// set value
// save secret
// unlock
