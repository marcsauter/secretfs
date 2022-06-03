package sekretsfs_test

import (
	"log"
	"os"
	"testing"

	"github.com/marcsauter/sekretsfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset *kubernetes.Clientset
)

func TestMain(m *testing.M) {
	ac, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		log.Fatal(err)
	}

	rc, err := clientcmd.NewNonInteractiveClientConfig(*ac, "kind-kind", nil, nil).ClientConfig()
	if err != nil {
		log.Fatal(err)
	}

	cs, err := kubernetes.NewForConfig(rc)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := cs.DiscoveryClient.ServerVersion(); err != nil {
		log.Fatalf("start kind first: %v", err)
	}

	clientset = cs

	exitVal := m.Run()

	os.Exit(exitVal)
}

func TestSekretsfsSecret(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := sekretsfs.New(clientset)
	require.NotNil(t, sfs)

	t.Run("Secret Mkdir and Remove", func(t *testing.T) {
		assert.NoError(t, sfs.Mkdir("default/testsecret", os.FileMode(0)))

		assert.Error(t, sfs.Mkdir("default/testsecret", os.FileMode(0)))

		assert.NoError(t, sfs.Remove("default/testsecret"))

		assert.Error(t, sfs.Remove("default/testsecret"))
	})

	t.Run("Secret Mkdir and RemoveAll", func(t *testing.T) {
		assert.NoError(t, sfs.Mkdir("default/testsecret1", os.FileMode(0)))

		assert.NoError(t, sfs.RemoveAll("default/testsecret1"))

		assert.NoError(t, sfs.RemoveAll("default/testsecret1"))
	})
}

func TestSekretsfsSecretKey(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := sekretsfs.New(clientset)
	require.NotNil(t, sfs)

	t.Run("Key Create and Remove", func(t *testing.T) {
		assert.NoError(t, sfs.Mkdir("default/testsecret1", os.FileMode(0)))

		f, err := sfs.Create("default/testsecret1/key1")
		assert.NoError(t, err)
		assert.NotNil(t, f)

		assert.NoError(t, sfs.Remove("default/testsecret1/key1"))

		assert.Error(t, sfs.Remove("default/testsecret1/key1"))

		assert.NoError(t, sfs.RemoveAll("default/testsecret1"))

	})
}
