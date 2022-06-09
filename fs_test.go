package sekretsfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFSName(t *testing.T) {
	sfs := New(nil)
	require.NotNil(t, sfs)

	assert.Equal(t, "SekretsFS", sfs.Name())
}
