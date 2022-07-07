package backend

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInternalExternalName(t *testing.T) {
	cs := NewFakeClientset()
	b := New(cs,
		WithSecretPrefix("unit-"),
		WithSecretSuffix("-test"),
	)

	externalName := "password"
	internalName := "unit-password-test"

	in := b.(*backend).internalName(externalName)
	require.Equal(t, internalName, in)

	en := b.(*backend).externalName(internalName)
	require.Equal(t, externalName, en)
}
