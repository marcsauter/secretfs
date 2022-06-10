package sekretsfs_test

import (
	"io/fs"
	"os"
	"path"
	"syscall"
	"testing"

	"github.com/marcsauter/sekretsfs"
	"github.com/marcsauter/sekretsfs/internal/backend"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFSName(t *testing.T) {
	sfs := sekretsfs.New(nil)
	require.NotNil(t, sfs)

	assert.Equal(t, "SekretsFS", sfs.Name())
}

func TestCreateOpen(t *testing.T) {
	namespace := "default"
	secret := "testsecret"
	key := "testitem"

	secretname := path.Join(namespace, secret)
	filename := path.Join(namespace, secret, key)

	sfs := sekretsfs.New(backend.NewFakeClientset())
	require.NotNil(t, sfs)

	t.Run("Create secret", func(t *testing.T) {
		f, err := sfs.Open(secretname)
		require.ErrorIs(t, err, syscall.ENOENT)
		require.Nil(t, f)

		err = sfs.Mkdir(filename, os.FileMode(0))
		require.ErrorIs(t, err, syscall.ENOTDIR)

		err = sfs.Mkdir(secretname, os.FileMode(0))
		require.NoError(t, err)

		err = sfs.Mkdir(secretname, os.FileMode(0))
		require.ErrorIs(t, err, afero.ErrFileExists)

		err = sfs.MkdirAll(secretname, os.FileMode(0))
		require.ErrorIs(t, err, afero.ErrFileExists)
	})

	t.Run("Open secret", func(t *testing.T) {
		f, err := sfs.Open(secretname)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface os.FileInfo
		st, err := f.Stat()
		require.NoError(t, err)

		require.Equal(t, secret, f.Name())
		require.Equal(t, int64(0), st.Size())
		require.Equal(t, fs.ModeDir, st.Mode())
		require.False(t, st.ModTime().IsZero())
		require.True(t, st.IsDir())
		// require.Equal(t, f, f.Sys())
	})

	t.Run("Create file", func(t *testing.T) {
		f, err := sfs.Open(filename)
		require.ErrorIs(t, err, syscall.ENOENT)
		require.Nil(t, f)

		f, err = sfs.Create(filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface os.FileInfo
		st, err := f.Stat()
		require.NoError(t, err)

		require.Equal(t, key, st.Name())
		require.Equal(t, int64(1), st.Size())
		require.Equal(t, fs.FileMode(0), st.Mode())
		require.False(t, st.ModTime().IsZero())
		require.False(t, st.IsDir())
		// require.Equal(t, st, st.Sys())
	})

	t.Run("Open file", func(t *testing.T) {
		f, err := sfs.Open(filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface os.FileInfo
		st, err := f.Stat()
		require.NoError(t, err)

		require.Equal(t, key, st.Name())
		require.Equal(t, int64(1), st.Size())
		require.Equal(t, fs.FileMode(0), st.Mode())
		require.False(t, st.ModTime().IsZero())
		require.False(t, st.IsDir())
		// require.Equal(t, st, st.Sys())
	})
}
