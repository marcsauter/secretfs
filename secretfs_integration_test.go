package secretfs_test

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/marcsauter/secretfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset *kubernetes.Clientset
)

func TestMain(m *testing.M) {
	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	c, err := clientcmd.BuildConfigFromFlags("", filepath.Join(u.HomeDir, ".kube", "config"))
	if err != nil {
		log.Fatal(err)
	}

	cs, err := kubernetes.NewForConfig(c)
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

func TestConnection(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := secretfs.New(clientset)
	require.NotNil(t, sfs)

	assert.Equal(t, "SecretFS", sfs.Name())

	var err error

	err = sfs.Mkdir("default/testsecret", os.FileMode(0))
	assert.NoError(t, err)

	err = sfs.Mkdir("default/testsecret", os.FileMode(0))
	assert.Error(t, err)

	err = sfs.RemoveAll("default/testsecret")
	assert.Error(t, err)
}
