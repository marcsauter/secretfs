package sekretsfs

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/marcsauter/sekretsfs/internal/backend"
	"github.com/spf13/afero"
)

// File is the corev1.File without k8s specific data
// TODO: locking - keep cascading locking mind
type File struct {
	name  string // absolute name namespace/secret[/key]
	spath *secretPath

	key   string
	value []byte
	data  map[string][]byte

	size  int64
	mtime time.Time
	mode  fs.FileMode

	readonly bool
	closed   bool

	TLS bool // TODO: corev1.SecretTypeTLS

	mu      sync.Mutex
	backend backend.Backend
}

func newFile(name string) (*File, error) {
	p, err := newSecretPath(name)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	if p.isDir {
		return nil, &fs.PathError{Op: "open", Path: name, Err: syscall.EISDIR}
	}

	return &File{
		name:     name,
		spath:    p,
		key:      p.Key(),
		readonly: true,
	}, nil
}

// FileOpen returns a secret item
// Secret is also afero.File and os.FileInfo
func FileOpen(b backend.Backend, name string) (*File, error) {
	s, err := newFile(name)
	if err != nil {
		return nil, err
	}

	if err := b.Get(s); err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	v, ok := s.data[s.key]
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: syscall.ENOENT}
	}

	s.value = v
	s.backend = b

	return s, nil
}

// FileCreate returns a new or truncated secret item
func FileCreate(b backend.Backend, name string) (*File, error) {
	s, err := newFile(name)
	if err != nil {
		return nil, err
	}

	if err := b.Get(s); err != nil {
		return nil, &fs.PathError{Op: "create", Path: name, Err: err}
	}

	// TODO: create with truncate only if o_creat

	if err := b.Update(s); err != nil {
		return nil, &fs.PathError{Op: "create", Path: name, Err: err}
	}

	s.readonly = false
	s.backend = b

	return s, nil
}

var _ backend.Secret = (*File)(nil)

// Namespace returns the namespace name
func (s *File) Namespace() string {
	return s.spath.Namespace()
}

// Secret returns the name of the secret
func (s *File) Secret() string {
	return s.spath.Secret()
}

// Key returns the file name
func (s *File) Key() string {
	return s.key
}

// Value returns the file content
func (s *File) Value() []byte {
	return s.value
}

// Data returns the underlying secret data map
func (s *File) Data() map[string][]byte {
	return s.data
}

// SetData sets the secret data map
func (s *File) SetData(data map[string][]byte) {
	s.data = data
	s.size = int64(len(s.data))
}

var _ afero.File = (*File)(nil)  // https://pkg.go.dev/github.com/spf13/afero#File
var _ os.FileInfo = (*File)(nil) // https://pkg.go.dev/io/fs#FileInfo

// Close io.Closer
func (s *File) Close() error {
	if s.closed {
		return afero.ErrFileClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.Sync(); err != nil {
		return err
	}

	s.closed = true

	return nil
}

// Read io.Reader
// https://pkg.go.dev/io#Reader
func (s *File) Read(p []byte) (n int, err error) {
	if s.spath.IsDir() {
		return 0, syscall.EISDIR
	}

	if s.closed {
		return 0, afero.ErrFileClosed
	}

	return bytes.NewReader(s.value).Read(p)
}

// ReadAt io.ReaderAt
// https://pkg.go.dev/io#ReaderAt
func (s *File) ReadAt(p []byte, off int64) (n int, err error) {
	if s.spath.IsDir() {
		return 0, syscall.EISDIR
	}

	if s.closed {
		return 0, afero.ErrFileClosed
	}

	return bytes.NewReader(s.value).Read(p)
}

// Seek io.Seeker
// https://pkg.go.dev/io#Seeker
func (s *File) Seek(offset int64, whence int) (int64, error) {
	if s.spath.IsDir() {
		return 0, syscall.EISDIR
	}

	if s.closed {
		return 0, afero.ErrFileClosed
	}

	return bytes.NewReader(s.value).Seek(offset, whence)
}

// Write io.Writer
// https://pkg.go.dev/io#Writer
func (s *File) Write(p []byte) (n int, err error) {
	if s.spath.IsDir() {
		return 0, syscall.EISDIR
	}

	if s.closed {
		return 0, afero.ErrFileClosed
	}

	b := bytes.NewBuffer(s.value)

	n, err = b.Write(p)
	if err != nil {
		return 0, err
	}

	s.mu.Lock()
	s.value = b.Bytes()
	s.mu.Unlock()

	return n, nil
}

// WriteAt io.WriterAt
// https://pkg.go.dev/io#WriterAt
// Source: https://github.com/aws/aws-sdk-go/blob/e8afe81156c70d5bf7b6d2ed5aeeb609ea3ba3f8/aws/types.go#L183
func (s *File) WriteAt(p []byte, off int64) (n int, err error) {
	if s.spath.IsDir() {
		return 0, syscall.EISDIR
	}

	if s.closed {
		return 0, afero.ErrFileClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pLen := len(p)
	expLen := off + int64(pLen)

	if int64(len(s.value)) < expLen {
		if int64(cap(s.value)) < expLen {
			buf := make([]byte, expLen)
			copy(buf, s.value)
			s.value = buf
		}

		s.value = s.value[:expLen]
	}

	copy(s.value[off:], p)

	return pLen, nil
}

// Name returns the secret or item name (afero.File, io.FileInfo)
func (s *File) Name() string {
	return s.spath.Name()
}

// Readdir (afero.File)
func (s *File) Readdir(count int) ([]os.FileInfo, error) {
	if !s.spath.IsDir() {
		return nil, &fs.PathError{Op: "read", Path: s.Name(), Err: syscall.ENOTDIR}
	}

	entries := []os.FileInfo{}

	for n := range s.data {
		p := &secretPath{
			namespace: s.spath.Namespace(),
			secret:    s.spath.Secret(),
			key:       n,
			isDir:     false,
		}

		entries = append(entries, &File{
			name:  p.Absolute(),
			spath: p,
		})

		if count > 0 && len(entries) == count {
			break
		}
	}

	return entries, nil
}

// Readdirnames (afero.File)
func (s *File) Readdirnames(n int) ([]string, error) {
	fi, err := s.Readdir(n)

	names := make([]string, len(fi))
	for i, f := range fi {
		_, names[i] = filepath.Split(f.Name())
	}

	return names, err
}

// Stat (afero.File)
func (s *File) Stat() (os.FileInfo, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Sync (afero.File)
func (s *File) Sync() error {
	if s.spath.IsDir() {
		return &fs.PathError{Op: "read", Path: s.Name(), Err: syscall.EISDIR}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	act := &File{}
	if err := s.backend.Get(act); err != nil {
		return err
	}

	act.data[s.key] = s.value

	return s.backend.Update(act)
}

// Truncate (afero.File)
func (s *File) Truncate(size int64) error {
	if s.spath.IsDir() {
		return &fs.PathError{Op: "read", Path: s.Name(), Err: syscall.EISDIR}
	}

	if int64(len(s.value)) <= size {
		return nil
	}

	s.value = append([]byte{}, s.value[:size]...)

	return nil
}

// WriteString (afero.File)
func (s *File) WriteString(st string) (int, error) {
	if s.spath.IsDir() {
		return 0, &fs.PathError{Op: "read", Path: s.Name(), Err: syscall.EISDIR}
	}

	return bytes.NewBuffer(s.value).WriteString(st)
}

// Size returns length in bytes for keys (io.FileInfo)
func (s *File) Size() int64 {
	return s.size
}

// Mode returns file mode bits (io.FileInfo)
func (s *File) Mode() fs.FileMode {
	return s.mode
}

// ModTime returns file modification time (io.FileInfo)
func (s *File) ModTime() time.Time {
	return s.mtime
}

// IsDir returns true for a secret, false for a key (io.FileInfo)
func (s *File) IsDir() bool {
	return s.spath.IsDir()
}

// Sys returns underlying data source (io.FileInfo)
// can return nil
func (s *File) Sys() interface{} {
	return s
}

func (s *File) isEmptyDir() bool {
	return s.spath.IsDir() && len(s.data) == 0
}

// TODO: checks, errors
func (s *File) deleteFile(name string) error {
	if _, ok := s.data[name]; !ok {
		return afero.ErrFileNotFound
	}

	delete(s.data, name)
	s.size = int64(len(s.data))
	s.mtime = time.Now()

	return nil
}

// TODO: checks, errors
func (s *File) renameFile(o, n string) {
	s.data[n] = s.data[o]
	delete(s.data, o)
	s.size = int64(len(s.data))
	s.mtime = time.Now()
}
