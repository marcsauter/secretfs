// Package secret provides a independent secret
package secret

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/marcsauter/sekretsfs/internal/io"
	"github.com/spf13/afero"
)

// Secret is the corev1.Secret without k8s specific data
// TODO: locking - keep cascading locking mind
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

	mu      sync.Mutex
	backend io.LoadStorer
}

// New returns a new Secret
// Secret is also afero.File and os.FileInfo
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

var _ io.Secreter = (*Secret)(nil)

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

var _ os.FileInfo = (*Secret)(nil) // https://pkg.go.dev/io/fs#FileInfo

// Close io.Closer
func (s *Secret) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s == nil {
		return os.ErrInvalid
	}

	defer func() {
		s = nil
	}()

	return s.Sync()
}

// Read io.Reader
// https://pkg.go.dev/io#Reader
func (s *Secret) Read(p []byte) (n int, err error) {
	return 0, syscall.EROFS
}

// ReadAt io.ReaderAt
// https://pkg.go.dev/io#ReaderAt
func (s *Secret) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, syscall.EROFS
}

// Seek io.Seeker
// https://pkg.go.dev/io#Seeker
func (s *Secret) Seek(offset int64, whence int) (int64, error) {
	return 0, syscall.EROFS
}

// Write io.Writer
// https://pkg.go.dev/io#Writer
func (s *Secret) Write(p []byte) (n int, err error) {
	return 0, syscall.EROFS
}

// WriteAt io.WriterAt
// https://pkg.go.dev/io#WriterAt
func (s *Secret) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, syscall.EROFS
}

// Name returns the secret name (afero.File, io.FileInfo)
func (s *Secret) Name() string {
	return filepath.Base(s.name)
}

// Readdir (afero.File)
func (s *Secret) Readdir(count int) ([]os.FileInfo, error) {
	return nil, &os.PathError{Op: "readdir", Path: s.Name(), Err: syscall.ENOTDIR}
}

// Readdirnames (afero.File)
func (s *Secret) Readdirnames(n int) ([]string, error) {
	fi, err := s.Readdir(n)

	names := make([]string, len(fi))
	for i, f := range fi {
		_, names[i] = filepath.Split(f.Name())
	}

	return names, err
}

// Stat (afero.File)
func (s *Secret) Stat() (os.FileInfo, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Sync (afero.File)
func (s *Secret) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	act := &Secret{}
	if err := s.backend.Load(act); err != nil {
		return err
	}

	s.SetData(act.Data())

	return s.backend.Store(s)
}

// Truncate (afero.File)
func (s *Secret) Truncate(size int64) error {
	return syscall.EROFS
}

// WriteString (afero.File)
func (s *Secret) WriteString(st string) (int, error) {
	return 0, syscall.EROFS
}

// Size returns length in bytes for keys (io.FileInfo)
func (s *Secret) Size() int64 {
	return s.size
}

// Mode returns file mode bits (io.FileInfo)
func (s *Secret) Mode() fs.FileMode {
	return s.mode
}

// ModTime returns file modification time (io.FileInfo)
func (s *Secret) ModTime() time.Time {
	return s.mtime
}

// IsDir returns true for a secret, false for a key (io.FileInfo)
func (s *Secret) IsDir() bool {
	return s.isDir
}

// Sys returns underlying data source (io.FileInfo)
// can return nil
func (s *Secret) Sys() interface{} {
	return s
}
