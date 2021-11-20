package secretfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAnnotaiion(t *testing.T) {
	t.Run("add", func(t *testing.T) {
		s := &corev1.Secret{}

		addAnnotation(s)
		require.NotNil(t, s.Annotations)

		v, ok := s.Annotations[AnnotationKey]
		assert.True(t, ok)
		assert.Equal(t, AnnotationValue, v)
	})

	t.Run("check valid", func(t *testing.T) {
		s := &corev1.Secret{}
		s.Annotations = map[string]string{
			AnnotationKey: AnnotationValue,
		}

		ok := checkAnnotaion(s)
		assert.True(t, ok)
	})

	t.Run("check valid", func(t *testing.T) {
		s := &corev1.Secret{}
		s.Annotations = map[string]string{
			AnnotationKey: "",
		}

		ok := checkAnnotaion(s)
		assert.False(t, ok)
	})
}
