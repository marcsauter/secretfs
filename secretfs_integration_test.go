package secretfs_test

import (
	"os"
	"testing"

	"github.com/marcsauter/secretfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	kubeconfig = "./testdata/.kubeconfig"
)

func TestConnection(t *testing.T) {
	t.SkipNow()
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	assert.NoError(t, err)
	require.NotNil(t, config)

	clientset, err := kubernetes.NewForConfig(config)
	assert.NoError(t, err)
	require.NotNil(t, clientset)

	sfs := secretfs.New(clientset)
	require.NotNil(t, sfs)

	assert.Equal(t, "SecretFS", sfs.Name())

	err = sfs.Mkdir("default/testsecret", os.FileMode(0))
	assert.NoError(t, err)

	err = sfs.Mkdir("default/testsecret", os.FileMode(0))
	assert.Error(t, err)

	err = sfs.RemoveAll("default/testsecret")
	assert.Error(t, err)
}
