package backend_test

import (
	"syscall"
	"testing"
	"time"

	"github.com/marcsauter/secfs/internal/backend"
	"github.com/stretchr/testify/require"
)

// TODO: test Delete()

func TestBackend(t *testing.T) {
	cs := backend.NewFakeClientset()
	b := backend.New(cs)

	t.Run("get secret not managed with secfs", func(t *testing.T) {
		s, err := newFakeSecret("default", "notmanaged", "", []byte{})
		require.NoError(t, err)

		err = b.Get(s)
		require.ErrorIs(t, err, backend.ErrNotManaged)
	})

	t.Run("create get", func(t *testing.T) {
		s, err := newFakeSecret("default", "secret", "", []byte{})
		require.NoError(t, err)

		data := map[string][]byte{
			"key1": []byte("value1"),
		}

		s.SetData(data)

		err = b.Create(s)
		require.NoError(t, err)

		s1, err := newFakeSecret("default", "secret", "", []byte{})
		require.NoError(t, err)

		err = b.Get(s1)
		require.NoError(t, err)
		require.Equal(t, data, s1.Data())
		require.Equal(t, 1, len(s1.Data()))
		require.Equal(t, []byte("value1"), s1.Data()["key1"])
	})

	t.Run("get update get", func(t *testing.T) {
		s, err := newFakeSecret("default", "secret", "key2", []byte("value2"))
		require.NoError(t, err)

		err = b.Get(s)
		require.NoError(t, err)

		err = b.Update(s)
		require.NoError(t, err)

		s1, err := newFakeSecret("default", "secret", "", []byte{})
		require.NoError(t, err)

		err = b.Get(s1)
		require.NoError(t, err)

		require.Equal(t, 2, len(s1.Data()))
		require.Equal(t, []byte("value1"), s1.Data()["key1"])
		require.Equal(t, []byte("value2"), s1.Data()["key2"])
	})

	t.Run("rename", func(t *testing.T) {
		// TODO:
		// rename old does not exist
		// rename new does already exist

		o, err := newFakeSecret("default", "secret-not-existing", "", []byte{})
		require.NoError(t, err)

		n, err := newFakeSecret("default", "secret-new", "", []byte{})
		require.NoError(t, err)

		err = b.Rename(o, n)
		require.ErrorIs(t, err, syscall.ENOENT)

		o, err = newFakeSecret("default", "secret", "", []byte{})
		require.NoError(t, err)

		n, err = newFakeSecret("default", "secret-existing", "", []byte{})
		require.NoError(t, err)

		err = b.Create(n)
		require.NoError(t, err)

		err = b.Rename(o, n)
		require.ErrorIs(t, err, syscall.EEXIST)

		o, err = newFakeSecret("default", "secret", "", []byte{})
		require.NoError(t, err)

		n, err = newFakeSecret("default", "secret-new", "", []byte{})
		require.NoError(t, err)

		err = b.Rename(o, n)
		require.NoError(t, err)

		err = b.Get(n)
		require.NoError(t, err)

		require.Equal(t, 2, len(n.Data()))
		require.Equal(t, []byte("value1"), n.Data()["key1"])
		require.Equal(t, []byte("value2"), n.Data()["key2"])
	})

	t.Run("delete get delete", func(t *testing.T) {
		s, err := newFakeSecret("default", "secret-new", "", []byte{})
		require.NoError(t, err)

		err = b.Delete(s)
		require.NoError(t, err)

		err = b.Get(s)
		require.ErrorIs(t, err, syscall.ENOENT)

		err = b.Delete(s)
		require.NoError(t, err)
	})
}

type fakeSecret struct {
	namespace string
	secret    string
	key       string
	value     []byte
	data      map[string][]byte
	mtime     time.Time
}

func newFakeSecret(ns, s, k string, v []byte) (backend.Secret, error) {
	return &fakeSecret{
		namespace: ns,
		secret:    s,
		key:       k,
		value:     v,
	}, nil
}

func (s *fakeSecret) Namespace() string {
	return s.namespace
}
func (s *fakeSecret) Secret() string {
	return s.secret
}
func (s *fakeSecret) Key() string {
	return s.key
}
func (s *fakeSecret) Value() []byte {
	return s.value
}

func (s *fakeSecret) Data() map[string][]byte {
	return s.data
}

func (s *fakeSecret) SetData(data map[string][]byte) {
	s.data = data
}

func (s *fakeSecret) SetTime(mtime time.Time) {
	s.mtime = mtime
}

func (s *fakeSecret) Delete() bool {
	return false
}
