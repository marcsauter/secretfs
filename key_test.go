package secretfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestKey(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		s := &corev1.Secret{}
		key := "key1"

		createKey(s, key)
		require.NotNil(t, s.Data)

		v, ok := s.Data[key]
		assert.True(t, ok)
		assert.Empty(t, v)

	})
}
