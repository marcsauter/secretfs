// Package secfs is a filesystem for k8s secrets
// Namespace -> directory
// Secret -> directory
// Secret key -> file
// Absolute path to secret key: namespace/secret/key
package secfs

import (
	"errors"
	"io/fs"
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
		return wrapPathError("Mkdir", name, err)
	}

	if !s.IsDir() {
		return wrapPathError("Mkdir", name, syscall.ENOTDIR)
	}

	_, err = Open(sfs.backend, name)

	if err == nil {
		return wrapPathError("Mkdir", name, syscall.EEXIST)
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return wrapPathError("Mkdir", name, err)
	}

	return wrapPathError("Mkdir", name, sfs.backend.Create(s))
}

// MkdirAll calls Mkdir
func (sfs secfs) MkdirAll(p string, perm os.FileMode) error {
	return sfs.Mkdir(p, perm)
}

// Open opens a file, returning it or an error, if any happens.
// https://pkg.go.dev/os#Open
func (sfs secfs) Open(name string) (afero.File, error) {
	return Open(sfs.backend, name)
}

// OpenFile opens a file using the given flags and the given mode.
// https://pkg.go.dev/os#OpenFile
// perm will be ignored because there is nothing comparable to filesystem permission for Kubernetes secrets
//nolint:gocognit,gocyclo // complex function
func (sfs secfs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	s, err := newFile(name)
	if err != nil {
		return nil, wrapPathError("OpenFile", name, err)
	}

	f, err := sfs.Open(name)

	// open a file read-only or open a directory
	if err == nil && (s.IsDir() || (flag == os.O_RDONLY)) {
		return f, nil
	}

	// Ensure that this call creates the file:
	// If O_EXCL is specified with O_CREAT, and pathname already exists, then  open() fails with the error EEXIST.
	if err == nil && (flag&os.O_EXCL > 0) && (flag&os.O_CREATE > 0) {
		return nil, wrapPathError("Mkdir", name, syscall.EEXIST)
	}

	// If pathname does not exist, create it as a regular file.
	if os.IsNotExist(err) && (flag&os.O_CREATE > 0) {
		f, err = sfs.Create(name)
	}

	// Handle unexpected error from Open and error from Create
	if err != nil {
		return nil, err
	}

	// enable read-write mode
	f.(*File).readonly = false

	if flag&os.O_APPEND > 0 {
		if _, err := f.Seek(0, os.SEEK_END); err != nil {
			return nil, err
		}
	}

	if flag&os.O_TRUNC > 0 && flag&(os.O_RDWR|os.O_WRONLY) > 0 {
		if err := f.Truncate(0); err != nil {
			return nil, err
		}
	}

	return f, nil
}

// Remove removes an empty secret or a key identified by name.
func (sfs secfs) Remove(name string) error {
	si, err := sfs.Stat(name)
	if err != nil {
		return wrapPathError("Remove", name, err)
	}

	s := si.Sys().(*File)

	if si.IsDir() {
		if !s.isEmptyDir() {
			return wrapPathError("Remove", name, syscall.ENOTEMPTY)
		}

		// remove empty secret
		if err := sfs.backend.Delete(s); err != nil {
			return wrapPathError("Remove", name, err)
		}

		return nil
	}

	// remove secret key
	s.delete = true

	return wrapPathError("Remove", name, sfs.backend.Update(s))
}

// RemoveAll removes a secret or key with all it contains.
// It does not fail if the path does not exist (return nil).
func (sfs secfs) RemoveAll(name string) error {
	si, err := sfs.Stat(name)
	if errors.Is(err, afero.ErrFileNotFound) {
		return nil
	}

	if err != nil {
		return wrapPathError("RemoveAll", name, err)
	}

	s := si.Sys().(*File)

	if si.IsDir() {
		// remove secret
		if err := sfs.backend.Delete(s); err != nil {
			return wrapPathError("RemoveAll", name, err)
		}

		return nil
	}

	// remove secret key
	s.delete = true

	return wrapPathError("RemoveAll", name, sfs.backend.Update(s))
}

// Rename moves old to new. Rename does not replace existing secrets or files.
func (sfs secfs) Rename(o, n string) error {
	oldSp, err := newSecretPath(o)
	if err != nil {
		return wrapLinkError("Rename", o, n, err)
	}

	newSp, err := newSecretPath(n)
	if err != nil {
		return wrapLinkError("Rename", o, n, err)
	}

	// move secret in a different namespace - currently not allowed
	// ns1/sec1 -> ns2/sec2
	// TODO: discuss
	if oldSp.Namespace() != newSp.Namespace() {
		return wrapLinkError("Rename", o, n, ErrMoveCrossNamespace)
	}

	// rename secret
	// sec1 -> sec2
	if oldSp.IsDir() {
		if newSp.IsDir() {
			return wrapLinkError("Rename", o, n, sfs.backend.Rename(oldSp, newSp))
		}

		return wrapLinkError("Rename", o, n, ErrMoveConvert)
	}

	// move/rename key
	ofi, err := Open(sfs.backend, o)
	if err != nil {
		return wrapLinkError("Rename", o, n, err)
	}

	// sec1/key1 -> sec2 // move key1 from sec1 to sec2 // sec2 must exist
	// sec1/key1 -> sec1/key2 // rename key1 to key2 - key2 will be replaced
	// sec1/key1 -> sec2/key2 // move key1 as key2 to sec2 // sec2 must exist, sec2/key2 will be replaced
	name := oldSp.Key()
	if !newSp.IsDir() {
		name = newSp.Key()
	}

	// create new item
	nfi, err := FileCreate(sfs.backend, path.Join(newSp.Namespace(), newSp.Secret(), name))
	if err != nil {
		return wrapLinkError("Rename", o, n, err)
	}

	nfi.value = ofi.value

	if err := nfi.Close(); err != nil {
		return wrapLinkError("Rename", o, n, err)
	}

	ofi.delete = true

	return wrapLinkError("Rename", o, n, sfs.backend.Update(ofi))
}

// Stat returns a FileInfo describing the named secret/key, or an error.
func (sfs secfs) Stat(name string) (os.FileInfo, error) {
	return Open(sfs.backend, name)
}

// Chmod changes the mode of the named file to mode.
func (sfs secfs) Chmod(name string, mode os.FileMode) error {
	return nil
}

// Chown changes the uid and gid of the named file.
func (sfs secfs) Chown(name string, uid, gid int) error {
	return nil
}

// Chtimes changes the access and modification times of the named file
func (sfs secfs) Chtimes(name string, atime, mtime time.Time) error {
	return nil
}
