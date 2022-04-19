package secret_test

import (
	"io/fs"
	"os"
	"testing"

	"github.com/marcsauter/sekretsfs/internal/secret"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecretAndAferoFileInfoInterface(t *testing.T) {
	s, err := secret.New("/default/testsecret")
	require.NoError(t, err)
	require.NotNil(t, s)

	assert.Equal(t, "default", s.Namespace())
	assert.Equal(t, "testsecret", s.Secret())

	assert.Equal(t, "testsecret", s.Name())
	assert.Empty(t, s.Size())
	assert.Equal(t, fs.ModeDir, s.Mode())
	assert.False(t, s.ModTime().IsZero())
	assert.True(t, s.IsDir())
	assert.Equal(t, s, s.Sys())
}

func TestNewSecretKeyAndAferoFileInfoInterface(t *testing.T) {
	s, err := secret.New("/default/testsecret/tls.crt")
	require.NoError(t, err)
	require.NotNil(t, s)

	assert.Equal(t, "default", s.Namespace())
	assert.Equal(t, "testsecret", s.Secret())

	assert.Equal(t, "tls.crt", s.Name())
	assert.Empty(t, s.Size())
	assert.Equal(t, os.FileMode(0), s.Mode())
	assert.False(t, s.ModTime().IsZero())
	assert.False(t, s.IsDir())
	assert.Equal(t, s, s.Sys())
}

func TestNewSecretInvalid(t *testing.T) {
	t.Run("invalid path 1", func(t *testing.T) {
		s, err := secret.New("/default")
		assert.Error(t, err)
		assert.Nil(t, s)
	})

	t.Run("invalid path 2", func(t *testing.T) {
		s, err := secret.New("/default/testsecret/key/more")
		assert.Error(t, err)
		assert.Nil(t, s)

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
		assert.Equal(t, int64(2), s.Size())

		act := s.Data()
		assert.Equal(t, exp, act)
	})

	t.Run("update get key", func(t *testing.T) {
		s, err := secret.New("/default/testsecret")
		require.NoError(t, err)
		require.NotNil(t, s)

		s.Update("key1", []byte("value1"))
		assert.Equal(t, int64(1), s.Size())

		v, ok := s.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", string(v))
	})

	t.Run("add get key", func(t *testing.T) {
		s, err := secret.New("/default/testsecret")
		require.NoError(t, err)
		require.NotNil(t, s)

		err = s.Add("key1", []byte("value1"))
		assert.NoError(t, err)
		assert.Equal(t, int64(1), s.Size())

		v, ok := s.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", string(v))

		err = s.Add("key1", []byte("value1"))
		assert.Error(t, err)
		assert.Equal(t, int64(1), s.Size())
	})

	t.Run("add update get key", func(t *testing.T) {
		s, err := secret.New("/default/testsecret")
		require.NoError(t, err)
		require.NotNil(t, s)

		err = s.Add("key1", []byte("value1"))
		assert.NoError(t, err)
		assert.Equal(t, int64(1), s.Size())

		v, ok := s.Get("key1")
		assert.True(t, ok)
		assert.Equal(t, "value1", string(v))

		err = s.Add("key2", []byte("value2"))
		assert.NoError(t, err)
		assert.Equal(t, int64(2), s.Size())

		v, ok = s.Get("key2")
		assert.True(t, ok)
		assert.Equal(t, "value2", string(v))

		s.Update("key3", []byte("value3"))
		assert.Equal(t, int64(3), s.Size())

		v, ok = s.Get("key3")
		assert.True(t, ok)
		assert.Equal(t, "value3", string(v))

		err = s.Add("key3", []byte("value3"))
		assert.Error(t, err)
		assert.Equal(t, int64(3), s.Size())

		s.Update("key3", []byte("value4"))
		assert.Equal(t, int64(3), s.Size())

		v, ok = s.Get("key3")
		assert.True(t, ok)
		assert.Equal(t, "value4", string(v))

		err = s.Delete("key3")
		assert.NoError(t, err)
		assert.Equal(t, int64(2), s.Size())

		err = s.Delete("key3")
		assert.ErrorIs(t, afero.ErrFileNotFound, err)
		assert.Equal(t, int64(2), s.Size())

		v, ok = s.Get("key3")
		assert.False(t, ok)
		assert.Nil(t, v)
	})
}
