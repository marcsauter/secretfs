package secfs_test

import (
	"context"
	"io/fs"
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

func TestSecfsSecret(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("Secret Mkdir and Remove", func(t *testing.T) {
		secretname := "default/testsecret1"

		require.NoError(t, sfs.Mkdir(secretname, os.FileMode(0)))

		require.ErrorIs(t, sfs.Mkdir(secretname, os.FileMode(0)), fs.ErrExist)

		require.NoError(t, sfs.Remove(secretname))

		require.ErrorIs(t, sfs.Remove(secretname), fs.ErrNotExist)

		f, err := sfs.Open(secretname)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Nil(t, f)
	})

	t.Run("Secret Mkdir and RemoveAll", func(t *testing.T) {
		secretname := "default/testsecret2"

		require.NoError(t, sfs.Mkdir(secretname, os.FileMode(0)))

		require.NoError(t, sfs.RemoveAll(secretname))

		require.NoError(t, sfs.RemoveAll(secretname))

		f, err := sfs.Open(secretname)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Nil(t, f)
	})
}

func TestSecfsFile(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("Key Create and Remove", func(t *testing.T) {
		secretname := "default/testsecret3"
		filename := path.Join(secretname, "file1")

		require.NoError(t, sfs.Mkdir(secretname, os.FileMode(0)))

		f, err := sfs.Create(filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		require.ErrorIs(t, sfs.Remove(secretname), syscall.ENOTEMPTY)

		require.NoError(t, sfs.Remove(filename))

		require.ErrorIs(t, sfs.Remove(filename), fs.ErrNotExist)

		require.NoError(t, sfs.Remove(secretname))

		f, err = sfs.Open(secretname)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Nil(t, f)
	})

	t.Run("Key Create and RemoveAll", func(t *testing.T) {
		secretname := "default/testsecret4"
		filename := path.Join(secretname, "file1")

		require.NoError(t, sfs.Mkdir(secretname, os.FileMode(0)))

		f, err := sfs.Create(filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		require.NoError(t, sfs.RemoveAll(secretname), "remove all")

		f, err = sfs.Open(secretname)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Nil(t, f)
	})
}

func TestAferoFunctions(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("afero.WriteFile afero.ReadFile", func(t *testing.T) {
		secretname := "default/testsecret5"
		filename := path.Join(secretname, "file1")
		content := []byte("0123456789")

		require.NoError(t, sfs.Mkdir(secretname, 0o0700))

		err := afero.WriteFile(sfs, filename, content, 0o0600)
		require.NoError(t, err)

		c, err := afero.ReadFile(sfs, filename)
		require.NoError(t, err)
		require.Equal(t, content, c)

		require.NoError(t, sfs.Remove(filename))
		require.ErrorIs(t, sfs.Remove(filename), os.ErrNotExist)
		require.NoError(t, sfs.RemoveAll(secretname))
	})

	t.Run("afero.FileContainsBytes afero.FileContainsAnyBytes", func(t *testing.T) {
		secretname := "default/testsecret6"
		filename := path.Join(secretname, "file1")
		content := []byte("0123456789")

		require.NoError(t, sfs.Mkdir(secretname, 0o0700))

		err := afero.WriteFile(sfs, filename, content, 0o0600)
		require.NoError(t, err)

		ok, err := afero.FileContainsBytes(sfs, filename, []byte("123"))
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = afero.FileContainsBytes(sfs, filename, []byte("ABC"))
		require.NoError(t, err)
		require.False(t, ok)

		ok, err = afero.FileContainsAnyBytes(sfs, filename, [][]byte{
			[]byte("123"),
			[]byte("ABC"),
		},
		)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = afero.FileContainsAnyBytes(sfs, filename, [][]byte{
			[]byte("321"),
			[]byte("CBA"),
		},
		)
		require.NoError(t, err)
		require.False(t, ok)

		require.NoError(t, sfs.Remove(filename))
		require.NoError(t, sfs.RemoveAll(secretname))
	})

	t.Run("afero.DirExists afero.Exists afero.IsDir", func(t *testing.T) {
		secretname := "default/testsecret7"

		filename := path.Join(secretname, "file1")
		content := []byte("0123456789")

		ok, err := afero.Exists(sfs, secretname)
		require.NoError(t, err)
		require.False(t, ok)

		ok, err = afero.DirExists(sfs, secretname)
		require.NoError(t, err)
		require.False(t, ok)

		ok, err = afero.IsDir(sfs, secretname)
		require.ErrorIs(t, err, os.ErrNotExist)
		require.False(t, ok)

		require.NoError(t, sfs.Mkdir(secretname, 0o0700))

		ok, err = afero.Exists(sfs, secretname)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = afero.IsDir(sfs, secretname)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = afero.DirExists(sfs, secretname)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = afero.IsEmpty(sfs, secretname)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = afero.Exists(sfs, filename)
		require.NoError(t, err)
		require.False(t, ok)

		ok, err = afero.DirExists(sfs, filename)
		require.NoError(t, err)
		require.False(t, ok)

		err = afero.WriteFile(sfs, filename, []byte{}, 0o0600)
		require.NoError(t, err)

		ok, err = afero.Exists(sfs, filename)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = afero.DirExists(sfs, filename)
		require.NoError(t, err)
		require.False(t, ok)

		ok, err = afero.IsDir(sfs, filename)
		require.NoError(t, err)
		require.False(t, ok)

		ok, err = afero.IsEmpty(sfs, filename)
		require.NoError(t, err)
		require.True(t, ok)

		err = afero.WriteFile(sfs, filename, content, 0o0600)
		require.NoError(t, err)

		ok, err = afero.IsEmpty(sfs, filename)
		require.NoError(t, err)
		require.False(t, ok)

		require.NoError(t, sfs.RemoveAll(secretname))
	})

	t.Run("afero.FileContainsBytes and afero.FileContainsAnyBytes", func(t *testing.T) {
		secretname := "default/testsecret6"
		basename := "file1"
		filename := path.Join(secretname, basename)
		content := []byte("0123456789")

		require.NoError(t, sfs.Mkdir(secretname, 0o0700))

		bpfs := afero.NewBasePathFs(sfs, secretname)
		require.NotNil(t, bpfs)

		err := afero.WriteFile(bpfs, basename, content, 0o0600)
		require.NoError(t, err)

		p := afero.FullBaseFsPath(bpfs.(*afero.BasePathFs), basename)
		require.Equal(t, filename, p)

		c, err := afero.ReadFile(sfs, filename)
		require.NoError(t, err)
		require.Equal(t, content, c)

		require.NoError(t, sfs.RemoveAll(secretname))
	})
}

/*
TODO: tests
func GetTempDir(fs Fs, subPath string) string
func Glob(fs Fs, pattern string) (matches []string, err error)
func ReadAll(r io.Reader) ([]byte, error)
func ReadDir(fs Fs, dirname string) ([]os.FileInfo, error)
func SafeWriteReader(fs Fs, path string, r io.Reader) (err error)
func TempDir(fs Fs, dir, prefix string) (name string, err error)
func Walk(fs Fs, root string, walkFn filepath.WalkFunc) error
func WriteReader(fs Fs, path string, r io.Reader) (err error)
*/
