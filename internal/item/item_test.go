package item_test

import (
	"testing"

	"github.com/marcsauter/sekretsfs/internal/item"
	"github.com/marcsauter/sekretsfs/internal/secret"
	"github.com/stretchr/testify/require"
)

func TestNewItem(t *testing.T) {
	s, err := secret.New("/default/testsecret")
	require.NoError(t, err)
	require.NotNil(t, s)

	it := item.New(nil, s, "key1", []byte("value1")) // TODO: use fake backend
	require.NotNil(t, it)
}

func TestAferoFileInterface(t *testing.T) {
}
