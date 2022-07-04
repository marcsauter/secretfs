package secfs

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/marcsauter/secfs/internal/backend"
	"github.com/spf13/afero"
)

// File is the corev1.Secret without k8s specific data
// TODO: locking - keep cascading locking in mind
type File struct {
	name  string // absolute name namespace/secret[/key]
	spath *secretPath

	key   string
	value []byte
	data  map[string][]byte

	mtime time.Time
	mode  fs.FileMode

	readonly bool
	closed   bool
	delete   bool

	TLS bool // TODO: corev1.SecretTypeTLS

	mu      sync.Mutex
	backend backend.Backend
}

func newFile(name string) (*File, error) {
	p, err := newSecretPath(name)
	if err != nil {
		return nil, err
	}

	mode := os.FileMode(0)
	if p.IsDir() {
		mode = os.ModeDir
	}

	return &File{
		name:     name,
		spath:    p,
		key:      p.Key(),
		data:     make(map[string][]byte),
		mode:     mode,
		readonly: true,
	}, nil
}

// Open open a secret or file
// https://pkg.go.dev/os#Open
// returns *File (implements afero.File and os.FileInfo)
func Open(b backend.Backend, name string) (*File, error) {
	f, err := newFile(name)
	if err != nil {
		return nil, wrapPathError("Open", name, err)
	}

	f.backend = b
	f.readonly = true

	if err := b.Get(f); err != nil {
		return nil, wrapPathError("Open", name, err)
	}

	if f.IsDir() {
		return f, nil
	}

	v, ok := f.data[f.key]
	if !ok {
		return nil, wrapPathError("Open", name, syscall.ENOENT)
	}

	f.value = v

	return f, nil
}

// FileCreate create a new or truncated file
// https://pkg.go.dev/os#Create
// returns *File (implements afero.File and os.FileInfo)
func FileCreate(b backend.Backend, name string) (*File, error) {
	f, err := newFile(name)
	if err != nil {
		return nil, wrapPathError("Create", name, err)
	}

	f.backend = b
	f.readonly = false

	if f.IsDir() {
		return nil, wrapPathError("Create", name, syscall.EISDIR)
	}

	if err := b.Get(f); err != nil {
		return nil, wrapPathError("Create", name, syscall.ENOENT)
	}

	f.data[f.key] = make([]byte, 0)

	if err := b.Update(f); err != nil {
		return nil, wrapPathError("Create", name, err)
	}

	return f, nil
}

var _ backend.Secret = (*File)(nil) // backend.Secret includes backend.Metadata

// Namespace returns the namespace name (backend.Metadata)
func (f *File) Namespace() string {
	return f.spath.Namespace()
}

// Secret returns the name of the secret (backend.Metadata)
func (f *File) Secret() string {
	return f.spath.Secret()
}

// Key returns the file name (backend.Metadata)
func (f *File) Key() string {
	return f.key
}

// Value returns the file content (backend.Secret)
func (f *File) Value() []byte {
	return f.value
}

// Delete key (backend.Secret)
func (f *File) Delete() bool {
	return f.delete
}

// SetData sets the secret data map (backend.Secret)
func (f *File) SetData(data map[string][]byte) {
	f.data = data
}

// Data returns the underlying secret data map (backend.Secret)
func (f *File) Data() map[string][]byte {
	return f.data
}

// SetTime sets the secret mtime (backend.Secret)
func (f *File) SetTime(mtime time.Time) {
	f.mtime = mtime
}

var _ afero.File = (*File)(nil)  // https://pkg.go.dev/github.com/spf13/afero#File
var _ os.FileInfo = (*File)(nil) // https://pkg.go.dev/io/fs#FileInfo

// Close io.Closer
func (f *File) Close() error {
	if err := f.validateRO(); err != nil {
		return err
	}

	if err := f.Sync(); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.closed = true

	return nil
}

// Read io.Reader
// https://pkg.go.dev/io#Reader
func (f *File) Read(p []byte) (n int, err error) {
	if err := f.validateRO(); err != nil {
		return 0, err
	}

	return bytes.NewReader(f.value).Read(p)
}

// ReadAt io.ReaderAt
// https://pkg.go.dev/io#ReaderAt
func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	if err := f.validateRO(); err != nil {
		return 0, err
	}

	return bytes.NewReader(f.value).Read(p)
}

// Seek io.Seeker
// https://pkg.go.dev/io#Seeker
func (f *File) Seek(offset int64, whence int) (int64, error) {
	if err := f.validateRO(); err != nil {
		return 0, err
	}

	return bytes.NewReader(f.value).Seek(offset, whence)
}

// Write io.Writer
// https://pkg.go.dev/io#Writer
func (f *File) Write(p []byte) (n int, err error) {
	if err := f.validateRW(); err != nil {
		return 0, err
	}

	b := bytes.NewBuffer(f.value)

	// err is always nil
	// https://pkg.go.dev/bytes#Buffer.Write
	n, _ = b.Write(p)

	// b.Bytes() has b.Cap() > b.Len()
	// so we need to create the fitting byte slice and copy the content of the buffer
	v := make([]byte, b.Len())
	copy(v, b.Bytes())

	f.mu.Lock()
	f.value = v
	f.mu.Unlock()

	return n, nil
}

// WriteAt io.WriterAt
// https://pkg.go.dev/io#WriterAt
// Source: https://github.com/aws/aws-sdk-go/blob/e8afe81156c70d5bf7b6d2ed5aeeb609ea3ba3f8/aws/types.go#L183
func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	if err := f.validateRW(); err != nil {
		return 0, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	pLen := len(p)
	expLen := off + int64(pLen)

	if int64(len(f.value)) < expLen {
		if int64(cap(f.value)) < expLen {
			buf := make([]byte, expLen)
			copy(buf, f.value)
			f.value = buf
		}

		f.value = f.value[:expLen]
	}

	copy(f.value[off:], p)

	return pLen, nil
}

// Name returns the name of the secret or file (afero.File, io.FileInfo)
func (f *File) Name() string {
	return f.spath.Name()
}

// Readdir (afero.File)
// TODO: When return io.EOF?
func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	if !f.spath.IsDir() {
		return nil, syscall.ENOTDIR
	}

	entries := []os.FileInfo{}

	for n := range f.data {
		p := &secretPath{
			namespace: f.spath.Namespace(),
			secret:    f.spath.Secret(),
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
func (f *File) Readdirnames(n int) ([]string, error) {
	fi, err := f.Readdir(n)

	names := make([]string, len(fi))
	for i, f := range fi {
		_, names[i] = filepath.Split(f.Name())
	}

	return names, err
}

// Stat (afero.File)
func (f *File) Stat() (os.FileInfo, error) {
	return f, nil
}

// Sync (afero.File)
func (f *File) Sync() error {
	if err := f.validateRO(); err != nil {
		return err
	}

	if f.readonly {
		return nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.backend.Update(f)
}

// Truncate (afero.File)
func (f *File) Truncate(size int64) error {
	if err := f.validateRW(); err != nil {
		return wrapPathError("Truncate", f.name, err)
	}

	if int64(len(f.value)) <= size {
		return nil
	}

	f.value = append([]byte{}, f.value[:size]...)

	return nil
}

// WriteString (afero.File)
func (f *File) WriteString(st string) (int, error) {
	if err := f.validateRW(); err != nil {
		return 0, err
	}

	return bytes.NewBuffer(f.value).WriteString(st)
}

// Size returns length in bytes for keys (io.FileInfo)
func (f *File) Size() int64 {
	if f.spath.IsDir() {
		return int64(len(f.data))
	}

	return int64(len(f.value))
}

// Mode returns file mode bits (io.FileInfo)
func (f *File) Mode() fs.FileMode {
	return f.mode
}

// ModTime returns file modification time (io.FileInfo)
func (f *File) ModTime() time.Time {
	return f.mtime
}

// IsDir returns true for a secret, false for a key (io.FileInfo)
func (f *File) IsDir() bool {
	return f.spath.IsDir()
}

// Sys returns underlying data source (io.FileInfo)
// can return nil
func (f *File) Sys() interface{} {
	return f
}

func (f *File) isEmptyDir() bool {
	return f.spath.IsDir() && len(f.data) == 0
}

func (f *File) validateRO() error {
	if f.spath.IsDir() {
		return syscall.EISDIR
	}

	if f.closed {
		return afero.ErrFileClosed
	}

	return nil
}

func (f *File) validateRW() error {
	if err := f.validateRO(); err != nil {
		return err
	}

	if f.readonly {
		/*
			From the man page of truncate(2):
			EINVAL or EBADF
				The file descriptor fd is not open for writing.  POSIX
				permits, and portable applications should handle, either
				error for this case.  (Linux produces EINVAL.)

			We use EBADF which if more consistent here.
		*/
		return syscall.EBADF
	}

	return nil
}
