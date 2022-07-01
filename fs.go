// Package secfs is a filesystem for k8s secrets
// Namespace -> directory
// Secret -> directory
// Secret key -> file
// Absolute path to secret key: namespace/secret/key
package secfs

import (
	"errors"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/marcsauter/secfs/internal/backend"
	"github.com/spf13/afero"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

const (
	// DefaultSecretPrefix for k8s secrets
	DefaultSecretPrefix = ""
	// DefaultSecretSuffix for k8s secrets
	DefaultSecretSuffix = ""
	// DefaultRequestTimeout for k8s API requests
	DefaultRequestTimeout = 5 * time.Second
)

var (
	// ErrMoveCrossNamespace is currently not allowed
	ErrMoveCrossNamespace = errors.New("move a secret between namespaces is not allowed")
	// ErrMoveConvert secrets can contain files only
	ErrMoveConvert = errors.New("convert a secret to a file is not allowed")
)

// secfs implements afero.Fs for k8s secrets
type secfs struct {
	backend backend.Backend
	prefix  string
	suffix  string
	labels  map[string]string
	timeout time.Duration
	l       *zap.SugaredLogger
}

var _ afero.Fs = (*secfs)(nil) // https://pkg.go.dev/github.com/spf13/afero#Fs

// New returns a new afero.Fs for handling k8s secrets as files
func New(k kubernetes.Interface, opts ...Option) afero.Fs {
	s := &secfs{
		backend: backend.New(k),
		prefix:  DefaultSecretPrefix,
		suffix:  DefaultSecretSuffix,
		timeout: DefaultRequestTimeout,
		l:       zap.NewNop().Sugar(),
	}

	for _, option := range opts {
		option(s)
	}

	s.backend = backend.New(k,
		backend.WithSecretPrefix(s.prefix),
		backend.WithSecretSuffix(s.suffix),
		backend.WithSecretLabels(s.labels),
		backend.WithTimeout(s.timeout),
		backend.WithLogger(s.l),
	)

	return s
}

// Name of this FileSystem.
func (sfs secfs) Name() string {
	return "secfs"
}

// Create creates an key/value entry in the filesystem/secret
// returning the file/entry and an error, if any happens.
// https://pkg.go.dev/os#Create
func (sfs secfs) Create(name string) (afero.File, error) {
	return FileCreate(sfs.backend, name)
}

// Mkdir creates a new, empty secret
// return an error if any happens.
func (sfs secfs) Mkdir(name string, perm os.FileMode) error {
	s, err := newFile(name)
	if err != nil {
		return err
	}

	if !s.IsDir() {
		return syscall.ENOTDIR
	}

	_, err = Open(sfs.backend, name)

	if err == nil {
		return syscall.EEXIST
	}

	if !errors.Is(err, syscall.ENOENT) {
		return err
	}

	return sfs.backend.Create(s)
}

// MkdirAll calls Mkdir
func (sfs secfs) MkdirAll(p string, perm os.FileMode) error {
	return sfs.Mkdir(p, perm)
}

// Open opens a file, returning it or an error, if any happens.
// Open opens the named file for reading. If successful, methods on the returned file can be used for reading; the associated file descriptor has mode O_RDONLY. If there is an error, it will be of type *PathError.
func (sfs secfs) Open(name string) (afero.File, error) {
	return Open(sfs.backend, name) // TODO: add readonly
}

// OpenFile opens a file using the given flags and the given mode.
// OpenFile is the generalized open call; most users will use Open or Create instead. It opens the named file with specified flag (O_RDONLY etc.). If the file does not exist, and the O_CREATE flag is passed, it is created with mode perm (before umask). If successful, methods on the returned File can be used for I/O. If there is an error, it will be of type *PathError.
/*
perm &= chmodBits
chmod := false
file, err := m.openWrite(name)
if err == nil && (flag&os.O_EXCL > 0) {
	return nil, &os.PathError{Op: "open", Path: name, Err: ErrFileExists}
}
if os.IsNotExist(err) && (flag&os.O_CREATE > 0) {
	file, err = m.Create(name)
	chmod = true
}
if err != nil {
	return nil, err
}
if flag == os.O_RDONLY {
	file = mem.NewReadOnlyFileHandle(file.(*mem.File).Data())
}
if flag&os.O_APPEND > 0 {
	_, err = file.Seek(0, os.SEEK_END)
	if err != nil {
		file.Close()
		return nil, err
	}
}
if flag&os.O_TRUNC > 0 && flag&(os.O_RDWR|os.O_WRONLY) > 0 {
	err = file.Truncate(0)
	if err != nil {
		file.Close()
		return nil, err
	}
}
if chmod {
	return file, m.setFileMode(name, perm)
}
return file, nil
*/

func (sfs secfs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return sfs.Open(name) //TODO: implement/test
}

// Remove removes an empty secret or a key identified by name.
func (sfs secfs) Remove(name string) error {
	si, err := sfs.Stat(name)
	if err != nil {
		return err
	}

	s := si.Sys().(*File)

	if si.IsDir() {
		if !s.isEmptyDir() {
			return syscall.ENOTEMPTY
		}

		return sfs.backend.Delete(s) // remove empty secret
	}

	// remove secret key
	s.delete = true

	return sfs.backend.Update(s)
}

// RemoveAll removes a secret or key with all it contains.
// It does not fail if the path does not exist (return nil).
func (sfs secfs) RemoveAll(p string) error {
	si, err := sfs.Stat(p)
	if errors.Is(err, afero.ErrFileNotFound) {
		return nil
	}

	if err != nil {
		return err
	}

	s := si.Sys().(*File)

	if si.IsDir() {
		return sfs.backend.Delete(s) // remove secret
	}

	// remove secret key
	s.delete = true

	return sfs.backend.Update(s)
}

// Rename moves old to new. Rename does not replace existing secrets or files.
func (sfs secfs) Rename(o, n string) error {
	oldSp, err := newSecretPath(o)
	if err != nil {
		return &os.LinkError{Op: "rename", Old: o, New: n, Err: err}
	}

	newSp, err := newSecretPath(n)
	if err != nil {
		return &os.LinkError{Op: "rename", Old: o, New: n, Err: err}
	}

	// move secret in a different namespace - currently not allowed
	// ns1/sec1 -> ns2/sec2
	// TODO: discuss
	if oldSp.Namespace() != newSp.Namespace() {
		return &os.LinkError{Op: "rename", Old: o, New: n, Err: ErrMoveCrossNamespace}
	}

	// rename secret
	// sec1 -> sec2
	if oldSp.IsDir() {
		if newSp.IsDir() {
			return sfs.backend.Rename(oldSp, newSp)
		}

		return &os.LinkError{Op: "rename", Old: o, New: n, Err: ErrMoveConvert}
	}

	// move/rename key
	ofi, err := Open(sfs.backend, o)
	if err != nil {
		return &os.LinkError{Op: "rename", Old: o, New: n, Err: err}
	}

	// sec1/key1 -> sec2 // move key1 from sec1 to sec2 // sec2 must exist
	// sec1/key1 -> sec1/key2 // rename key1 to key2 - key2 will not be replaced
	// sec1/key1 -> sec2/key2 // move key1 as key2 to sec2 // sec2 must exist, sec2/key2 will be not replaced
	name := oldSp.Key()
	if !newSp.IsDir() {
		name = newSp.Key()
	}

	// create new item
	nfi, err := FileCreate(sfs.backend, path.Join(newSp.Namespace(), newSp.Secret(), name))
	if err != nil {
		return &os.LinkError{Op: "rename", Old: o, New: n, Err: err}
	}

	nfi.value = ofi.value

	if err := nfi.Sync(); err != nil {
		return &os.LinkError{Op: "rename", Old: o, New: n, Err: err}
	}

	// delete old item
	ofi.delete = true

	return ofi.Sync()
}

// Stat returns a FileInfo describing the named secret/key, or an error.
func (sfs secfs) Stat(name string) (os.FileInfo, error) {
	return Open(sfs.backend, name)
}

// Chmod changes the mode of the named file to mode.
// NOT IMPLEMENTED
func (sfs secfs) Chmod(name string, mode os.FileMode) error {
	return syscall.EROFS
}

// Chown changes the uid and gid of the named file.
// NOT IMPLEMENTED
func (sfs secfs) Chown(name string, uid, gid int) error {
	return syscall.EROFS
}

// Chtimes changes the access and modification times of the named file
// NOT IMPLEMENTED
func (sfs secfs) Chtimes(name string, atime, mtime time.Time) error {
	return syscall.EROFS
}
