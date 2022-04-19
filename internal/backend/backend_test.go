package backend_test

import (
	"testing"

	"github.com/marcsauter/sekretsfs/internal/backend"
	"github.com/marcsauter/sekretsfs/internal/secret"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestBackend(t *testing.T) {
	c := fake.NewSimpleClientset(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "notmanaged",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
	})

	t.Run("load secret not managed with sekretsfs", func(t *testing.T) {
		b := backend.New(c)

		s, err := secret.New("default/notmanaged")
		require.NoError(t, err)

		err = b.Load(s)
		assert.EqualError(t, err, "not managed with sekretsfs")
	})

	t.Run("store new and load", func(t *testing.T) {
		b := backend.New(c)

		s, err := secret.New("default/secret")
		require.NoError(t, err)

		data := map[string][]byte{
			"key1": []byte("value1"),
		}

		s.SetData(data)

		err = b.Store(s)
		assert.NoError(t, err)

		s1, err := secret.New("default/secret")
		require.NoError(t, err)

		err = b.Load(s1)
		assert.NoError(t, err)

		d1 := s.Data()
		assert.Equal(t, data, d1)
		assert.Equal(t, 1, len(d1))
		assert.Equal(t, []byte("value1"), d1["key1"])
	})

	t.Run("load change and store existing", func(t *testing.T) {
		b := backend.New(c)

		s, err := secret.New("default/secret")
		require.NoError(t, err)

		err = b.Load(s)
		assert.NoError(t, err)

		s.Add("key2", []byte("value2"))
		err = b.Store(s)
		assert.NoError(t, err)

		s1, err := secret.New("default/secret")
		require.NoError(t, err)

		err = b.Load(s1)
		assert.NoError(t, err)

		d1 := s1.Data()
		assert.Equal(t, 2, len(d1))
		assert.Equal(t, []byte("value1"), d1["key1"])
		assert.Equal(t, []byte("value2"), d1["key2"])
	})

	t.Run("delete and load", func(t *testing.T) {
		b := backend.New(c)

		s, err := secret.New("default/secret")
		require.NoError(t, err)

		err = b.Delete(s)
		assert.NoError(t, err)

		s1, err := secret.New("default/secret")
		require.NoError(t, err)

		err = b.Load(s1)
		assert.EqualError(t, err, "file does not exist")
	})
}
