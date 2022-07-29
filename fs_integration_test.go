package secfs_test

import (
	"context"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
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

func TestAferoFunctionsReadWrite(t *testing.T) {
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
}

func TestAferoFunctionsContains(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("afero.FileContainsBytes", func(t *testing.T) {
		secretname := "default/testsecret6a"
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

		require.NoError(t, sfs.RemoveAll(secretname))
	})

	t.Run("afero.FileContainsBytes", func(t *testing.T) {
		secretname := "default/testsecret6b"
		filename := path.Join(secretname, "file1")
		content := []byte("0123456789")

		require.NoError(t, sfs.Mkdir(secretname, 0o0700))

		err := afero.WriteFile(sfs, filename, content, 0o0600)
		require.NoError(t, err)

		ok, err := afero.FileContainsAnyBytes(sfs, filename, [][]byte{
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

		require.NoError(t, sfs.RemoveAll(secretname))
	})
}

func TestAferoFunctionsExists(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)
	secretname := "default/testsecret7"

	t.Run("afero.Exists afero.DirExists afero.IsDir all false", func(t *testing.T) {
		ok, err := afero.Exists(sfs, secretname)
		require.NoError(t, err)
		require.False(t, ok)

		ok, err = afero.DirExists(sfs, secretname)
		require.NoError(t, err)
		require.False(t, ok)

		ok, err = afero.IsDir(sfs, secretname)
		require.ErrorIs(t, err, os.ErrNotExist)
		require.False(t, ok)
	})

	t.Run("afero.Exists afero.DirExists afero.IsDir afero.IsEmpty on existing empty secret", func(t *testing.T) {
		require.NoError(t, sfs.Mkdir(secretname, 0o0700))

		ok, err := afero.Exists(sfs, secretname)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = afero.DirExists(sfs, secretname)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = afero.IsDir(sfs, secretname)
		require.NoError(t, err)
		require.True(t, ok)

		ok, err = afero.IsEmpty(sfs, secretname)
		require.NoError(t, err)
		require.True(t, ok)
	})

	defer func() {
		require.NoError(t, sfs.RemoveAll(secretname))
	}()

	t.Run("afero.Exists afero.DirExists afero.IsDir on not existing file", func(t *testing.T) {
		filename := path.Join(secretname, "file1")

		ok, err := afero.Exists(sfs, filename)
		require.NoError(t, err)
		require.False(t, ok)

		ok, err = afero.DirExists(sfs, filename)
		require.NoError(t, err)
		require.False(t, ok)

		ok, err = afero.IsDir(sfs, filename)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.False(t, ok)
	})

	t.Run("afero.Exists afero.DirExists afero.IsDir afero.IsEmpty on existing empty file", func(t *testing.T) {
		filename := path.Join(secretname, "file1")

		err := afero.WriteFile(sfs, filename, []byte{}, 0o0600)
		require.NoError(t, err)

		ok, err := afero.Exists(sfs, filename)
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
	})

	t.Run("afero.IsEmpty on existing non empty file", func(t *testing.T) {
		filename := path.Join(secretname, "file1")
		content := []byte("0123456789")

		err := afero.WriteFile(sfs, filename, content, 0o0600)
		require.NoError(t, err)

		ok, err := afero.IsEmpty(sfs, filename)
		require.NoError(t, err)
		require.False(t, ok)
	})
}

func TestAferoFunctionsBasePathFs(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("afero.NewBasePathFs and afero.FullBaseFsPath", func(t *testing.T) {
		secretname := "default/testsecret8"
		basename := "file1"
		filename := path.Join(secretname, basename)
		content := []byte("0123456789")

		require.NoError(t, sfs.Mkdir(secretname, 0o0700))

		bpFs := afero.NewBasePathFs(sfs, secretname)
		require.NotNil(t, bpFs)

		err := afero.WriteFile(bpFs, basename, content, 0o0600)
		require.NoError(t, err)

		p := afero.FullBaseFsPath(bpFs.(*afero.BasePathFs), basename)
		require.Equal(t, filename, p)

		c, err := afero.ReadFile(sfs, filename)
		require.NoError(t, err)

		require.Equal(t, content, c)

		require.NoError(t, sfs.RemoveAll(secretname))
	})
}

func TestAferoFunctionsGlob(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("afero.Glob", func(t *testing.T) {
		secretname := "default/testsecret9"
		secrets := []string{
			filepath.Join(secretname, "a"),
			filepath.Join(secretname, "b"),
			filepath.Join(secretname, "c"),
			filepath.Join(secretname, "d"),
			filepath.Join(secretname, "e"),
		}

		sort.Strings(secrets)

		require.NoError(t, sfs.Mkdir(secretname, 0o0700))

		for _, s := range secrets {
			_, err := sfs.Create(s)
			require.NoError(t, err)
		}

		result, err := afero.Glob(sfs, "default/testsecret9/*")
		require.NoError(t, err)

		sort.Strings(result)

		require.Equal(t, secrets, result)

		require.NoError(t, sfs.RemoveAll(secretname))
	})
}

func TestAferoFunctionsReadAll(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("afero.ReadAll", func(t *testing.T) {
		secretname := "default/testsecret10"
		filename := path.Join(secretname, "file1")
		content := []byte("0123456789")

		require.NoError(t, sfs.Mkdir(secretname, 0o0700))

		err := afero.WriteFile(sfs, filename, content, 0o0600)
		require.NoError(t, err)

		f, err := sfs.Open(filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		c, err := afero.ReadAll(f)
		require.NoError(t, err)
		require.Equal(t, content, c)
	})
}

func TestAferoFunctionsReadDir(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("afero.ReadDir", func(t *testing.T) {
		secretname := "default/testsecret11"
		secrets := []string{
			filepath.Join(secretname, "a"),
			filepath.Join(secretname, "b"),
			filepath.Join(secretname, "c"),
			filepath.Join(secretname, "d"),
			filepath.Join(secretname, "e"),
		}

		sort.Strings(secrets)

		require.NoError(t, sfs.Mkdir(secretname, 0o0700))

		for _, s := range secrets {
			_, err := sfs.Create(s)
			require.NoError(t, err)
		}

		fi, err := afero.ReadDir(sfs, secretname)
		require.NoError(t, err)

		result := make([]string, len(fi))

		for i, s := range fi {
			result[i] = filepath.Join(secretname, s.Name())
		}

		sort.Strings(result)

		require.Equal(t, secrets, result)

		require.NoError(t, sfs.RemoveAll(secretname))
	})
}

func TestAferoFunctionsWriteReader(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("afero.SafeWriteReader afero.WriteReader", func(t *testing.T) {
		secretname := "default/testsecret12"
		filenameR := filepath.Join(secretname, "read")
		filenameW := filepath.Join(secretname, "write")
		content := []byte("0123456789")

		require.NoError(t, sfs.Mkdir(secretname, 0o0700))

		err := afero.WriteFile(sfs, filenameR, content, 0o0600)
		require.NoError(t, err)

		// 1st
		f, err := sfs.Open(filenameR)
		require.NoError(t, err)
		require.NotNil(t, f)

		err = afero.WriteReader(sfs, filenameW, f)
		require.NoError(t, err)

		require.NoError(t, f.Close())

		// check
		c, err := afero.ReadFile(sfs, filenameW)
		require.NoError(t, err)
		require.Equal(t, content, c)

		// 2nd
		f, err = sfs.Open(filenameR)
		require.NoError(t, err)
		require.NotNil(t, f)

		err = afero.WriteReader(sfs, filenameW, f)
		require.NoError(t, err)

		require.NoError(t, f.Close())

		// 3rd but safe
		f, err = sfs.Open(filenameR)
		require.NoError(t, err)
		require.NotNil(t, f)

		err = afero.SafeWriteReader(sfs, filenameW, f)
		require.ErrorContains(t, err, "already exists")

		require.NoError(t, f.Close())

		require.NoError(t, sfs.RemoveAll(secretname))
	})
}

func TestAferoFunctionsTempDir(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("afero.TempDir", func(t *testing.T) {
		tmp, err := afero.TempDir(sfs, "default", "testsecret12")
		require.NoError(t, err)
		require.NotEmpty(t, tmp)

		fi, err := sfs.Stat(tmp)
		require.NoError(t, err)
		require.NotNil(t, fi)
		require.True(t, fi.IsDir())

		require.NoError(t, sfs.RemoveAll(tmp))

		tmp, err = afero.TempDir(sfs, "default/testsecret12", "testsecret12a")
		require.ErrorIs(t, err, syscall.ENOTDIR)
		require.Empty(t, tmp)
	})
}

func TestAferoFunctionsWalk(t *testing.T) {
	if clientset == nil {
		t.Skip("no cluster connection available")
	}

	sfs := testFs(t)

	t.Run("afero.Walk", func(t *testing.T) {
		secretname := "default/testsecret13"

		files := []string{"a", "b", "c", "d", "e"}

		exp := []string{}

		require.NoError(t, sfs.Mkdir(secretname, 0o0700))
		for _, f := range files {
			n := filepath.Join(secretname, f)
			_, err := sfs.Create(n)
			require.NoError(t, err)

			exp = append(exp, n)
		}

		t.Logf("created %d files", len(exp))

		act := []string{}

		err := afero.Walk(sfs, secretname, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			act = append(act, p)

			return nil
		})

		t.Logf("found %d files", len(exp))

		require.NoError(t, err)
		require.Equal(t, exp, act)

		require.NoError(t, sfs.RemoveAll(secretname))
	})
}
