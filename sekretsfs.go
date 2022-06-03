// Package sekretsfs is a filesystem for k8s secrets
// Namespace -> directory
// Secret -> directory
// Secret key -> file
// Absolute path to secret key: namespace/secret/key
package sekretsfs

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/marcsauter/sekretsfs/internal/backend"
	"github.com/marcsauter/sekretsfs/internal/io"
	"github.com/marcsauter/sekretsfs/internal/item"
	"github.com/marcsauter/sekretsfs/internal/secret"
	"github.com/spf13/afero"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

const (
	// DefaultSecretPrefix
	DefaultSecretPrefix = ""
	// DefaultSecretSuffix
	DefaultSecretSuffix = ""
	// DefaultRequestTimeout for k8s API requests
	DefaultRequestTimeout = 5 * time.Second
)

// sekretsFs implements afero.Fs for k8s secrets
type sekretsFs struct {
	backend io.LoadStoreDeleter
	prefix  string
	suffix  string
	timeout time.Duration
	l       *zap.SugaredLogger
}

var _ afero.Fs = (*sekretsFs)(nil)

// New returns a new afero.Fs for handling k8s secrets as files
func New(k kubernetes.Interface, opts ...Option) afero.Fs {
	s := &sekretsFs{
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
		backend.WithTimeout(s.timeout),
		backend.WithLogger(s.l),
	)

	return s
}

// Create creates an key/value entry in the filesystem/secret
// returning the file/entry and an error, if any happens.
// https://pkg.go.dev/os#Create
func (sfs sekretsFs) Create(name string) (afero.File, error) {
	si, err := sfs.Stat(name)
	if err == nil {
		if si.IsDir() {
			return nil, fmt.Errorf("%s is a secret", name)
		}
	}

	s := si.Sys().(*secret.Secret)

	s.Update(si.Name(), nil)

	return item.New(s, si.Name(), nil), sfs.backend.Store(s)
}

// Mkdir creates a new, empty secret
// return an error if any happens.
func (sfs sekretsFs) Mkdir(name string, perm os.FileMode) error {
	s, err := sfs.Stat(name)
	if errors.Is(err, afero.ErrFileNotFound) {
		if !s.IsDir() {
			return fmt.Errorf("%s is not a secret", name)
		}

		return sfs.backend.Store(s.Sys().(*secret.Secret))
	}

	if err == nil {
		return afero.ErrFileExists
	}

	return err
}

// MkdirAll calls Mkdir
func (sfs sekretsFs) MkdirAll(path string, perm os.FileMode) error {
	return sfs.Mkdir(path, perm)
}

// Open opens a file, returning it or an error, if any happens.
func (sfs sekretsFs) Open(name string) (afero.File, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// OpenFile opens a file using the given flags and the given mode.
func (sfs sekretsFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, fmt.Errorf("not yet implemented")
}

// Remove removes an empty secret or a key identified by name.
func (sfs sekretsFs) Remove(name string) error {
	si, err := sfs.Stat(name)
	if err != nil {
		return err
	}

	s := si.Sys().(*secret.Secret)

	if si.IsDir() {
		if len(s.Data()) != 0 {
			return fmt.Errorf("secret is not empty")
		}

		return sfs.backend.Delete(s) // remove empty secret
	}

	// remove secret key
	if err := s.Delete(si.Name()); err != nil {
		return err
	}

	return sfs.backend.Store(s)
}

// RemoveAll removes a secret or key with all it contains.
// It does not fail if the path does not exist (return nil).
func (sfs sekretsFs) RemoveAll(path string) error {
	si, err := sfs.Stat(path)
	if errors.Is(err, afero.ErrFileNotFound) {
		return nil
	}

	if err != nil {
		return err
	}

	s := si.Sys().(*secret.Secret)

	if si.IsDir() {
		return sfs.backend.Delete(s) // remove secret
	}

	// remove secret key
	_ = s.Delete(si.Name())

	return sfs.backend.Store(s)
}

// Rename renames (moves) old to new. If new already exists and is not a directory, Rename replaces it.
func (sfs sekretsFs) Rename(o, n string) error {
	osi, err := sfs.Stat(o)
	if err != nil {
		return err
	}

	oldSecret := osi.Sys().(*secret.Secret)

	nsi, err := sfs.Stat(n)
	if err != nil {
		return err
	}

	newSecret := nsi.Sys().(*secret.Secret)

	// ns1/sec1 -> ns2/sec2
	if oldSecret.Namespace() != newSecret.Namespace() {
		return errors.New("move a secret in a different namespaces is not allowed") // TODO: discuss
	}

	// move/rename secret
	// sec1 -> sec2
	if osi.IsDir() {
		if nsi.IsDir() { // ???
			return afero.ErrDestinationExists
		}

		newSecret.SetData(oldSecret.Data())

		if err := sfs.backend.Store(newSecret); err != nil {
			return err
		}

		return sfs.backend.Delete(oldSecret)
	}

	// move/rename key
	// sec1/key1 -> sec2 // move key1 from sec1 to sec2 // sec2 must exist
	// sec1/key1 -> sec1/key2 // rename key
	// sec1/key1 -> sec2/key2 // move key1 as key2 to sec2 // sec2 must exist, sec2/key2 will be replaced

	if !nsi.IsDir() { // ???
		return afero.ErrFileNotFound
	}

	if oldSecret.Path() == newSecret.Path() {
		return nil
	}

	v, ok := oldSecret.Get(osi.Name())
	if !ok {
		return afero.ErrFileNotFound
	}

	newSecret.Update(nsi.Name(), v)

	if err := sfs.backend.Store(newSecret); err != nil {
		return err
	}

	if err := oldSecret.Delete(osi.Name()); err != nil {
		return nil
	}

	return sfs.backend.Store(oldSecret)
}

// Stat returns a FileInfo describing the named secret/key, or an error.
func (sfs sekretsFs) Stat(name string) (os.FileInfo, error) {
	s, err := secret.New(name)
	if err != nil {
		return nil, err
	}

	if err := sfs.backend.Load(s); err != nil {
		return s, err
	}

	return s, nil
}

// Name of this FileSystem.
func (sfs sekretsFs) Name() string {
	return "SekretsFS"
}

// Chmod changes the mode of the named file to mode.
// NOT IMPLEMENTED
func (sfs sekretsFs) Chmod(name string, mode os.FileMode) error {
	return nil
}

// Chown changes the uid and gid of the named file.
// NOT IMPLEMENTED
func (sfs sekretsFs) Chown(name string, uid, gid int) error {
	return nil
}

// Chtimes changes the access and modification times of the named file
func (sfs sekretsFs) Chtimes(name string, atime, mtime time.Time) error {
	return nil
}
