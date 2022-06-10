package secfs_test

import (
	"context"
	"log"
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/marcsauter/secfs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace = "default"
	testLabel     = "integrationtest"
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

	// remove orphaned secrets
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secrets, err := cs.CoreV1().Secrets(testNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: testLabel,
	})
	if err != nil {
		log.Fatalf("failed to get secrets: %v", err)
	}

	for _, s := range secrets.Items {
		if err := cs.CoreV1().Secrets(testNamespace).Delete(ctx, s.Name, metav1.DeleteOptions{}); err != nil {
			log.Fatalf("failed to delete secret: %v", err)
		}
	}

	exitVal := m.Run()

	os.Exit(exitVal)
}

func testFs(t *testing.T) afero.Fs {
	sfs := secfs.New(clientset, secfs.WithSecretLabels(map[string]string{
		testLabel: "",
	}))
	require.NotNil(t, sfs)

	return sfs
}

func TestsecfsSecret(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("Secret Mkdir and Remove", func(t *testing.T) {
		secretname := "default/testsecret1"

		require.NoError(t, sfs.Mkdir(secretname, os.FileMode(0)))

		require.ErrorIs(t, sfs.Mkdir(secretname, os.FileMode(0)), syscall.EEXIST)

		require.NoError(t, sfs.Remove(secretname))

		require.ErrorIs(t, sfs.Remove(secretname), syscall.ENOENT)

		f, err := sfs.Open(secretname)
		require.ErrorIs(t, err, syscall.ENOENT)
		require.Nil(t, f)
	})

	t.Run("Secret Mkdir and RemoveAll", func(t *testing.T) {
		secretname := "default/testsecret2"

		require.NoError(t, sfs.Mkdir(secretname, os.FileMode(0)))

		require.NoError(t, sfs.RemoveAll(secretname))

		require.NoError(t, sfs.RemoveAll(secretname))

		f, err := sfs.Open(secretname)
		require.ErrorIs(t, err, syscall.ENOENT)
		require.Nil(t, f)
	})
}

func TestsecfsFile(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("Key Create and Remove", func(t *testing.T) {
		secretname := "default/testsecret3"
		filename := path.Join(secretname, "key1")

		require.NoError(t, sfs.Mkdir(secretname, os.FileMode(0)))

		f, err := sfs.Create(filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		require.ErrorIs(t, sfs.Remove(secretname), syscall.ENOTEMPTY)

		require.NoError(t, sfs.Remove(filename))

		require.ErrorIs(t, sfs.Remove(filename), syscall.ENOENT)

		require.NoError(t, sfs.Remove(secretname))

		f, err = sfs.Open(secretname)
		require.ErrorIs(t, err, syscall.ENOENT)
		require.Nil(t, f)
	})

	t.Run("Key Create and RemoveAll", func(t *testing.T) {
		secretname := "default/testsecret4"
		filename := path.Join(secretname, "key1")

		require.NoError(t, sfs.Mkdir(secretname, os.FileMode(0)))

		f, err := sfs.Create(filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		require.NoError(t, sfs.RemoveAll(secretname), "remove all")

		f, err = sfs.Open(secretname)
		require.ErrorIs(t, err, syscall.ENOENT)
		require.Nil(t, f)
	})
}
