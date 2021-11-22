package secretfs

import (
	"os"

	"github.com/spf13/afero"
	corev1 "k8s.io/api/core/v1"
)

// secretEntry is a k8s secret key/value entry implementing the afero.File interface
type secretEntry struct {
	secret *corev1.Secret
	key    string
	value  []byte
}

var _ afero.File = (*secretEntry)(nil)

// Close io.Closer
func (s secretEntry) Close() error {
	return nil
}

// Read io.Reader
func (s secretEntry) Read(p []byte) (n int, err error) {
	return
}

// ReadAt io.ReaderAt
func (s secretEntry) ReadAt(p []byte, off int64) (n int, err error) {
	return
}

// Seek io.Seeker
func (s secretEntry) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

// Write io.Writer
func (s secretEntry) Write(p []byte) (n int, err error) {
	return
}

// WriteAt io.WriterAt
func (s secretEntry) WriteAt(p []byte, off int64) (n int, err error) {
	return
}

// Name returns the secret name
func (s secretEntry) Name() string {
	return s.key
}

// Readdir afero.File
func (s secretEntry) Readdir(count int) ([]os.FileInfo, error) {
	return []os.FileInfo{}, nil
}

// Readdirnames afero.File
func (s secretEntry) Readdirnames(n int) ([]string, error) {
	return []string{}, nil
}

// Stat afero.File
func (s secretEntry) Stat() (os.FileInfo, error) {
	return nil, nil
}

// Sync afero.File
func (s secretEntry) Sync() error {
	return nil
}

// Truncate afero.File
func (s secretEntry) Truncate(size int64) error {
	return nil
}

// WriteString afero.File
func (s secretEntry) WriteString(st string) (ret int, err error) {
	return
}
