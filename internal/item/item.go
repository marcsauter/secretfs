// Package item represent an entry in the map of a Kubernetes secret
// item implements afero.File
package item

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/marcsauter/sekretsfs/internal/io"
	"github.com/marcsauter/sekretsfs/internal/secret"
	"github.com/spf13/afero"
)

// item is a k8s secret key/value entry implementing the afero.File interface
type item struct {
	key   string
	value []byte

	mu      sync.Mutex
	ref     *secret.Secret
	backend io.LoadStorer
	closed  bool
}

// New returns a new secret item
func New(b io.LoadStorer, s *secret.Secret, key string, value []byte) afero.File {
	return &item{
		backend: b,
		ref:     s,
		key:     key,
		value:   value,
	}
}

var _ afero.File = (*item)(nil)  // https://pkg.go.dev/github.com/spf13/afero#File
var _ os.FileInfo = (*item)(nil) // https://pkg.go.dev/io/fs#FileInfo

// Close io.Closer
func (i *item) Close() error {
	if i.closed {
		return afero.ErrFileClosed
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if i.ref == nil {
		return os.ErrInvalid
	}

	defer func() {
		i.ref = nil
	}()

	return i.Sync()
}

// Read io.Reader
// https://pkg.go.dev/io#Reader
func (i *item) Read(p []byte) (n int, err error) {
	if i.closed {
		return 0, afero.ErrFileClosed
	}

	return bytes.NewReader(i.value).Read(p)
}

// ReadAt io.ReaderAt
// https://pkg.go.dev/io#ReaderAt
func (i *item) ReadAt(p []byte, off int64) (n int, err error) {
	if i.closed {
		return 0, afero.ErrFileClosed
	}

	return bytes.NewReader(i.value).ReadAt(p, off)
}

// Seek io.Seeker
// https://pkg.go.dev/io#Seeker
func (i *item) Seek(offset int64, whence int) (int64, error) {
	if i.closed {
		return 0, afero.ErrFileClosed
	}

	return bytes.NewReader(i.value).Seek(offset, whence)
}

// Write io.Writer
// https://pkg.go.dev/io#Writer
func (i *item) Write(p []byte) (n int, err error) {
	if i.closed {
		return 0, afero.ErrFileClosed
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	return bytes.NewBuffer(i.value).Write(p)
}

// WriteAt io.WriterAt
// https://pkg.go.dev/io#WriterAt
// Source: https://github.com/aws/aws-sdk-go/blob/e8afe81156c70d5bf7b6d2ed5aeeb609ea3ba3f8/aws/types.go#L183
func (i *item) WriteAt(p []byte, off int64) (n int, err error) {
	if i.closed {
		return 0, afero.ErrFileClosed
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	pLen := len(p)
	expLen := off + int64(pLen)

	if int64(len(i.value)) < expLen {
		if int64(cap(i.value)) < expLen {
			buf := make([]byte, expLen)
			copy(buf, i.value)
			i.value = buf
		}

		i.value = i.value[:expLen]
	}

	copy(i.value[off:], p)

	return pLen, nil
}

// Name returns the secret name (afero.File, io.FileInfo)
func (i *item) Name() string {
	return i.key
}

// Readdir (afero.File)
// returns always an error, item can not be a directory
func (i *item) Readdir(count int) ([]os.FileInfo, error) {
	return nil, &os.PathError{Op: "readdir", Path: i.Name(), Err: syscall.ENOTDIR}
}

// Readdirnames (afero.File)
func (i *item) Readdirnames(n int) ([]string, error) {
	fi, err := i.Readdir(n)

	names := make([]string, len(fi))
	for i, f := range fi {
		_, names[i] = filepath.Split(f.Name())
	}

	return names, err
}

// Stat (afero.File)
func (i *item) Stat() (os.FileInfo, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Sync (afero.File)
func (i *item) Sync() error {
	if i.closed {
		return afero.ErrFileClosed
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if err := i.backend.Load(i.ref); err != nil {
		return err
	}

	if err := i.ref.Add(i.key, i.value); err != nil {
		return err
	}

	return i.backend.Store(i.ref)
}

// Truncate (afero.File)
func (i *item) Truncate(size int64) error {
	if i.closed {
		return afero.ErrFileClosed
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	if int64(len(i.value)) <= size {
		return nil
	}

	i.value = append([]byte{}, i.value[:size]...)

	return nil
}

// WriteString (afero.File)
func (i *item) WriteString(s string) (int, error) {
	if i.closed {
		return 0, afero.ErrFileClosed
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	return bytes.NewBuffer(i.value).WriteString(s)
}

// Size returns the size in bytes (io.FileInfo)
func (i *item) Size() int64 {
	return int64(len(i.value))
}

// Mode returns file mode bits (io.FileInfo)
func (i *item) Mode() fs.FileMode {
	return 0o666 // TODO:
}

// ModTime returns file modification time (io.FileInfo)
func (i *item) ModTime() time.Time {
	return time.Now() // TODO:
}

// IsDir always false for an Item (io.FileInfo)
func (i *item) IsDir() bool {
	return false
}

// Sys returns underlying data source (io.FileInfo)
// can return nil
func (i *item) Sys() interface{} {
	return nil // TODO:
}
