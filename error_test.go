package secfs

import (
	"io/fs"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorWrapper(t *testing.T) {
	t.Run("os.PathError", func(t *testing.T) {
		err := wrapPathError("testop", "testname", syscall.EISDIR)
		require.IsType(t, &os.PathError{}, err)
		require.Equal(t, &os.PathError{Op: "testop", Path: "testname", Err: syscall.EISDIR}, err.(*os.PathError))

		require.NoError(t, wrapPathError("test", "test", nil))
		require.ErrorIs(t, wrapPathError("test", "test", syscall.EEXIST), fs.ErrExist)
		require.ErrorIs(t, wrapPathError("test", "test", syscall.ENOENT), fs.ErrNotExist)
		require.ErrorIs(t, wrapPathError("test", "test", syscall.EISDIR), syscall.EISDIR)
	})

	t.Run("os.LinkError", func(t *testing.T) {
		err := wrapLinkError("testop", "oldname", "newname", syscall.EISDIR)
		require.IsType(t, &os.LinkError{}, err)
		require.Equal(t, &os.LinkError{Op: "testop", Old: "oldname", New: "newname", Err: syscall.EISDIR}, err.(*os.LinkError))

		require.NoError(t, wrapLinkError("test", "old", "new", nil))
		require.ErrorIs(t, wrapLinkError("test", "old", "new", syscall.EEXIST), fs.ErrExist)
		require.ErrorIs(t, wrapLinkError("test", "old", "new", syscall.ENOENT), fs.ErrNotExist)
		require.ErrorIs(t, wrapLinkError("test", "old", "new", syscall.EISDIR), syscall.EISDIR)

	})
}
