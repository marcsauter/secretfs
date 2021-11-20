// Package secretfs implements the handling of secrets for k8s
// Namespace -> directory
// Secret -> directory
// Secret key -> file
// Absolute path to secret key: namespace/secret/key
package secretfs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/spf13/afero"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// DefaultSecretPrefix for each secret name
	DefaultSecretPrefix = "" //nolint:gosec // no credentials
	// DefaultSecretSuffix for each secret name
	DefaultSecretSuffix = ""
	// DefaultRequestTimeout for k8s requests
	DefaultRequestTimeout = 5 * time.Second
)

// secretFs implements afero.secretFs for k8s secrets
type secretFs struct {
	c       *kubernetes.Clientset
	timeout time.Duration
	prefix  string
	suffix  string
	l       *zap.SugaredLogger
}

var _ afero.Fs = secretFs{}

// New returns a new afero.Fs for handling k8s secrets as files
func New(c *kubernetes.Clientset, opts ...Option) afero.Fs {
	s := &secretFs{
		c:       c,
		timeout: DefaultRequestTimeout,
		prefix:  DefaultSecretPrefix,
		suffix:  DefaultSecretSuffix,
		l:       zap.NewNop().Sugar(),
	}

	for _, option := range opts {
		option(s)
	}

	return s
}

func (sfs secretFs) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), sfs.timeout)
}

// Create creates an key/value entry in the filesystem/secret
// returning the file/entry and an error, if any happens.
func (sfs secretFs) Create(name string) (afero.File, error) {
	si, err := sfs.Stat(name)
	if err != nil {
		return nil, err
	}

	if si.IsDir() {
		return nil, fmt.Errorf("%s is a directory", name) // TODO: standard error
	}

	sec := si.Sys().(*corev1.Secret)
	sec.Data[si.Name()] = []byte{}

	ctx, cancel := sfs.context()
	defer cancel()

	resp, err := sfs.c.CoreV1().Secrets(sec.Namespace).Create(ctx, sec, metav1.CreateOptions{})

	return &secretEntry{
		secret: resp,
		key:    si.Name(),
		value:  []byte{},
	}, err
}

// Mkdir creates a new, empty secret
// return an error if any happens.
func (sfs secretFs) Mkdir(name string, perm os.FileMode) error {
	p, err := NewPath(name)
	if err != nil {
		return err
	}

	if !p.IsDir() {
		return fmt.Errorf("%s is not a directory/secret", name)
	}

	si, err := sfs.Stat(name)
	if err != nil && err != afero.ErrFileNotFound {
		return err
	}

	if si != nil {
		return afero.ErrFileExists
	}

	ctx, cancel := sfs.context()
	defer cancel()

	req := &corev1.Secret{}
	req.Name = p[SECRET]

	_, err = sfs.c.CoreV1().Secrets(p[NAMESPACE]).Create(ctx, req, metav1.CreateOptions{})

	return err
}

// MkdirAll does the same as Mkdir
func (sfs secretFs) MkdirAll(path string, perm os.FileMode) error {
	return sfs.Mkdir(path, perm)
}

// Open opens a file, returning it or an error, if any happens.
func (sfs secretFs) Open(name string) (afero.File, error) {
	return nil, nil
}

// OpenFile opens a file using the given flags and the given mode.
func (sfs secretFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return nil, nil
}

// Remove removes a file identified by name, returning an error, if any
// happens.
func (sfs secretFs) Remove(name string) error {
	_, err := sfs.Stat(name)
	if err != nil {
		return err
	}

	// file or dir

	_, cancel := sfs.context()
	defer cancel()

	return nil
}

// RemoveAll removes a directory path and any children it contains. It
// does not fail if the path does not exist (return nil).
func (sfs secretFs) RemoveAll(path string) error {
	_, err := sfs.Stat(path)
	if err != nil {
		return err
	}

	// file or dir

	_, cancel := sfs.context()
	defer cancel()

	return nil
}

// Rename renames (moves) oldpath to newpath. If newpath already exists and is not a directory, Rename replaces it.
func (sfs secretFs) Rename(oldname, newname string) error {
	osi, err := os.Stat(oldname)
	if errors.Is(err, os.ErrNotExist) {
		return afero.ErrFileNotFound
	}

	osec := osi.Sys().(corev1.Secret)

	nsi, err := os.Stat(newname)
	if err == nil && nsi.IsDir() {
		return afero.ErrDestinationExists
	}

	nsec := &corev1.Secret{}
	if nsi != nil {
		nsec = nsi.Sys().(*corev1.Secret)
	}

	if nsec.Namespace != "" && osec.Namespace != nsec.Namespace {
		return errors.New("move a secret in a different namespaces is not allowed") // TODO: discuss
	}

	ctx, cancel := sfs.context()
	defer cancel()

	// sec1/key1 -> sec1/key2 // rename key
	// sec1 -> sec2 // rename secret
	// sec1/key1 -> sec2 // move key1 from sec1 to sec2 // sec2 must exist
	// sec1/key1 -> sec2/key2 // move key1 as key2 to sec2 // sec2 must exist, sec2/key2 will be replaced

	_, err = sfs.c.CoreV1().Secrets(osec.Namespace).Update(ctx, nil, metav1.UpdateOptions{})

	return err
}

// Stat returns a FileInfo describing the named secret/key, or an error.
func (sfs secretFs) Stat(name string) (os.FileInfo, error) {
	fi := &secretInfo{
		name: name,
	}

	p, err := NewPath(name)
	if err != nil {
		return nil, err
	}

	switch len(p) {
	case 2:
		fi.isDir = true
		fi.mode = fs.FileMode(fs.ModeDir)
	case 3:
		fi.isDir = false
		fi.mode = fs.FileMode(0)
	default:
		return nil, fmt.Errorf("invalid path: %s", name)
	}

	ctx, cancel := sfs.context()
	defer cancel()

	s, err := sfs.c.CoreV1().Secrets(p[NAMESPACE]).Get(ctx, p[SECRET], metav1.GetOptions{})
	if err == nil {
		fi.mtime = s.CreationTimestamp.Time
		fi.size = int64(len(s.Data))
		fi.secret = s

		return fi, nil
	}

	if apierr.IsNotFound(err) {
		return nil, afero.ErrFileNotFound
	}

	return nil, err
}

// Name of this FileSystem.
func (sfs secretFs) Name() string {
	return "SecretFS"
}

// Chmod changes the mode of the named file to mode.
// NOT IMPLEMENTED
func (sfs secretFs) Chmod(name string, mode os.FileMode) error {
	return nil
}

// Chown changes the uid and gid of the named file.
// NOT IMPLEMENTED
func (sfs secretFs) Chown(name string, uid, gid int) error {
	return nil
}

// Chtimes changes the access and modification times of the named file
func (sfs secretFs) Chtimes(name string, atime, mtime time.Time) error {
	return nil
}
