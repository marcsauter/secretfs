package secfs

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

var (
	// ErrMoveCrossNamespace is currently not allowed
	ErrMoveCrossNamespace = errors.New("move a secret between namespaces is not allowed")
	// ErrMoveConvert secrets can contain files only
	ErrMoveConvert = errors.New("convert a secret to a file is not allowed")
)

func wrapPathError(op, name string, err error) error {
	switch err {
	case nil:
		return nil
	case syscall.EEXIST:
		return &os.PathError{Op: op, Path: name, Err: fs.ErrExist}
	case syscall.ENOENT:
		return &os.PathError{Op: op, Path: name, Err: fs.ErrNotExist}
	default:
		return &os.PathError{Op: op, Path: name, Err: err}
	}
}

//nolint:unparam // op is currently only "Rename"
func wrapLinkError(op, o, n string, err error) error {
	switch err {
	case nil:
		return nil
	case syscall.EEXIST:
		return &os.LinkError{Op: op, Old: o, New: n, Err: fs.ErrExist}
	case syscall.ENOENT:
		return &os.LinkError{Op: op, Old: o, New: n, Err: fs.ErrNotExist}
	default:
		return &os.LinkError{Op: op, Old: o, New: n, Err: err}
	}
}
