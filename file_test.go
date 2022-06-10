package sekretsfs_test

import (
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/marcsauter/sekretsfs"
	"github.com/marcsauter/sekretsfs/internal/backend"
	"github.com/stretchr/testify/require"
)

// TODO: add afero.File tests

func TestFileInterfaces(t *testing.T) {
	namespace := "default"
	secret := "testsecret"
	key := "testfile"

	filename := path.Join(namespace, secret, key)
	secretname := path.Join(namespace, secret)

	cs := backend.NewFakeClientset()

	// prepare
	sfs := sekretsfs.New(cs)
	err := sfs.Mkdir(secretname, os.FileMode(0))
	require.NoError(t, err)

	b := backend.New(cs)

	t.Run("FileCreate", func(t *testing.T) {
		f, err := sekretsfs.FileCreate(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface backend.Metadata
		require.Equal(t, namespace, f.Namespace())
		require.Equal(t, secret, f.Secret())
		require.Equal(t, key, f.Key())

		// interface os.FileInfo
		require.Equal(t, key, f.Name())
		require.Equal(t, int64(1), f.Size())
		require.Equal(t, fs.FileMode(0), f.Mode())
		require.False(t, f.ModTime().IsZero())
		require.False(t, f.IsDir())
		require.Equal(t, f, f.Sys())

		require.NoError(t, f.Close())
	})

	t.Run("Open file", func(t *testing.T) {
		f, err := sekretsfs.Open(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface backend.Metadata
		require.Equal(t, namespace, f.Namespace())
		require.Equal(t, secret, f.Secret())
		require.Equal(t, key, f.Key())

		// interface os.FileInfo
		require.Equal(t, key, f.Name())
		require.Equal(t, int64(1), f.Size())
		require.Equal(t, fs.FileMode(0), f.Mode())
		require.False(t, f.ModTime().IsZero())
		require.False(t, f.IsDir())
		require.Equal(t, f, f.Sys())
	})

	t.Run("Open secret", func(t *testing.T) {
		f, err := sekretsfs.Open(b, secretname)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface backend.Metadata
		require.Equal(t, namespace, f.Namespace())
		require.Equal(t, secret, f.Secret())
		require.Empty(t, f.Key())

		// interface os.FileInfo
		require.Equal(t, secret, f.Name())
		require.Equal(t, int64(1), f.Size())
		require.Equal(t, fs.ModeDir, f.Mode())
		require.False(t, f.ModTime().IsZero())
		require.True(t, f.IsDir())
		require.Equal(t, f, f.Sys())
	})
}

/*
func TestNewSecretKeyAndAferoFileInfoInterface(t *testing.T) {
	s, err := sekretsfs.FileOpen("/default/testsecret/tls.crt")
	require.NoError(t, err)
	require.NotNil(t, s)

	require.Equal(t, "default", s.Namespace())
	require.Equal(t, "testsecret", s.Secret())

	require.Equal(t, "tls.crt", s.Name())
	require.Empty(t, s.Size())
	require.Equal(t, os.FileMode(0), s.Mode())
	require.False(t, s.ModTime().IsZero())
	require.False(t, s.IsDir())
	require.Equal(t, s, s.Sys())
}

func TestNewSecretInvalid(t *testing.T) {
	t.Run("invalid path 1", func(t *testing.T) {
		s, err := secret.New("/default")
		require.Error(t, err)
		require.Nil(t, s)
	})

	t.Run("invalid path 2", func(t *testing.T) {
		s, err := secret.New("/default/testsecret/key/more")
		require.Error(t, err)
		require.Nil(t, s)

	})
}

func TestSecretCRUD(t *testing.T) {

	t.Run("set/get data source and size", func(t *testing.T) {
		s, err := secret.New("/default/testsecret")
		require.NoError(t, err)
		require.NotNil(t, s)

		exp := map[string][]byte{
			"key1": nil,
			"key2": nil,
		}

		s.SetData(exp)
		require.Equal(t, int64(2), s.Size())

		act := s.Data()
		require.Equal(t, exp, act)
	})

	t.Run("update get key", func(t *testing.T) {
		s, err := secret.New("/default/testsecret")
		require.NoError(t, err)
		require.NotNil(t, s)

		s.Update("key1", []byte("value1"))
		require.Equal(t, int64(1), s.Size())

		v, ok := s.Get("key1")
		require.True(t, ok)
		require.Equal(t, "value1", string(v))
	})

	t.Run("add get key", func(t *testing.T) {
		s, err := secret.New("/default/testsecret")
		require.NoError(t, err)
		require.NotNil(t, s)

		err = s.Add("key1", []byte("value1"))
		require.NoError(t, err)
		require.Equal(t, int64(1), s.Size())

		v, ok := s.Get("key1")
		require.True(t, ok)
		require.Equal(t, "value1", string(v))

		err = s.Add("key1", []byte("value1"))
		require.Error(t, err)
		require.Equal(t, int64(1), s.Size())
	})

	t.Run("add update get key", func(t *testing.T) {
		s, err := secret.New("/default/testsecret")
		require.NoError(t, err)
		require.NotNil(t, s)

		err = s.Add("key1", []byte("value1"))
		require.NoError(t, err)
		require.Equal(t, int64(1), s.Size())

		v, ok := s.Get("key1")
		require.True(t, ok)
		require.Equal(t, "value1", string(v))

		err = s.Add("key2", []byte("value2"))
		require.NoError(t, err)
		require.Equal(t, int64(2), s.Size())

		v, ok = s.Get("key2")
		require.True(t, ok)
		require.Equal(t, "value2", string(v))

		s.Update("key3", []byte("value3"))
		require.Equal(t, int64(3), s.Size())

		v, ok = s.Get("key3")
		require.True(t, ok)
		require.Equal(t, "value3", string(v))

		err = s.Add("key3", []byte("value3"))
		require.Error(t, err)
		require.Equal(t, int64(3), s.Size())

		s.Update("key3", []byte("value4"))
		require.Equal(t, int64(3), s.Size())

		v, ok = s.Get("key3")
		require.True(t, ok)
		require.Equal(t, "value4", string(v))

		err = s.Delete("key3")
		require.NoError(t, err)
		require.Equal(t, int64(2), s.Size())

		err = s.Delete("key3")
		require.ErrorIs(t, afero.ErrFileNotFound, err)
		require.Equal(t, int64(2), s.Size())

		v, ok = s.Get("key3")
		require.False(t, ok)
		require.Nil(t, v)
	})
}
*/
