package secfs_test

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/postfinance/secfs"
	"github.com/postfinance/secfs/internal/backend"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFSName(t *testing.T) {
	sfs := secfs.New(nil)
	require.NotNil(t, sfs)

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "secfs", sfs.Name())
	})

	t.Run("Unimplemented", func(t *testing.T) {
		namespace := "default"
		secret := "testsecret"

		secretname := path.Join(namespace, secret)

		require.NoError(t, sfs.Chmod(secretname, 0o0777))
		require.NoError(t, sfs.Chown(secretname, 0, 0))
		require.NoError(t, sfs.Chtimes(secretname, time.Now(), time.Now()))
	})
}

func TestFSCreate(t *testing.T) {
	namespace := "default"
	secret := "testsecret"
	key := "testfile"

	secretname := path.Join(namespace, secret)
	filename := path.Join(namespace, secret, key)

	sfs := secfs.New(backend.NewFakeClientset())
	require.NotNil(t, sfs)

	t.Run("Create secret", func(t *testing.T) {
		f, err := sfs.Open(secretname)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Nil(t, f)

		err = sfs.Mkdir(filename, os.FileMode(0))
		require.ErrorIs(t, err, syscall.ENOTDIR)

		err = sfs.Mkdir(secretname, os.FileMode(0))
		require.NoError(t, err)

		err = sfs.Mkdir(secretname, os.FileMode(0))
		require.ErrorIs(t, err, afero.ErrFileExists)

		err = sfs.MkdirAll(secretname, os.FileMode(0))
		require.NoError(t, err)
	})

	t.Run("Create file", func(t *testing.T) {
		f, err := sfs.Open(filename)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Nil(t, f)

		f, err = sfs.Create(filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface os.FileInfo
		st, err := f.Stat()
		require.NoError(t, err)

		require.Equal(t, key, st.Name())
		require.Equal(t, int64(0), st.Size())
		require.Equal(t, fs.FileMode(0), st.Mode())
		require.False(t, st.ModTime().IsZero())
		require.False(t, st.IsDir())
		// require.Equal(t, st, st.Sys())
	})
}

func TestFSOpen(t *testing.T) {
	namespace := "default"
	secret := "testsecret"
	key := "testfile"

	secretname := path.Join(namespace, secret)
	filename := path.Join(namespace, secret, key)

	sfs := secfs.New(backend.NewFakeClientset())
	require.NotNil(t, sfs)

	err := sfs.Mkdir(secretname, os.FileMode(0))
	require.NoError(t, err)

	f, err := sfs.Create(filename)
	require.NoError(t, err)
	require.NotNil(t, f)

	t.Run("Open secret", func(t *testing.T) {
		f, err := sfs.Open(secretname)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface os.FileInfo
		st, err := f.Stat()
		require.NoError(t, err)

		require.Equal(t, secret, f.Name())
		require.Equal(t, int64(1), st.Size())
		require.Equal(t, fs.ModeDir, st.Mode())
		require.False(t, st.ModTime().IsZero())
		require.True(t, st.IsDir())
		// require.Equal(t, f, f.Sys())
	})

	t.Run("Open file", func(t *testing.T) {
		f, err := sfs.Open(filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface os.FileInfo
		st, err := f.Stat()
		require.NoError(t, err)

		require.Equal(t, key, st.Name())
		require.Equal(t, int64(0), st.Size())
		require.Equal(t, fs.FileMode(0), st.Mode())
		require.False(t, st.ModTime().IsZero())
		require.False(t, st.IsDir())
		// require.Equal(t, st, st.Sys())
	})
}
func TestFSOpenFile(t *testing.T) {
	namespace := "default"
	secret := "testsecret"
	key := "testfile"

	secretname := path.Join(namespace, secret)
	filename := path.Join(namespace, secret, key)

	sfs := secfs.New(backend.NewFakeClientset())
	require.NotNil(t, sfs)

	err := sfs.Mkdir(secretname, os.FileMode(0))
	require.NoError(t, err)

	f, err := sfs.Create(filename)
	require.NoError(t, err)
	require.NotNil(t, f)

	t.Run("OpenFile secret", func(t *testing.T) {
		f, err := sfs.OpenFile(secretname, os.O_RDWR, 0o0777)
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err := f.Write([]byte{})
		require.Zero(t, n)
		require.ErrorIs(t, err, syscall.EISDIR)
	})

	t.Run("OpenFile file", func(t *testing.T) {
		const (
			value1 = "0123456789"
			value2 = "ABCDE"
		)

		// open file read-only
		f, err := sfs.OpenFile(filename, os.O_RDONLY, 0o0000)
		require.NotNil(t, f)
		require.NoError(t, err)

		n, err := f.Write([]byte(value1))
		require.Zero(t, n)
		require.ErrorIs(t, err, syscall.EBADF)

		require.NoError(t, f.Close())

		// open existing file with O_CREATE and O_EXCL
		f, err = sfs.OpenFile(filename, os.O_CREATE|os.O_EXCL, 0o0644)
		require.Nil(t, f)
		require.ErrorIs(t, err, fs.ErrExist)

		// new filename
		filename1 := path.Join(namespace, secret, fmt.Sprintf("%s1", key))

		// open not existing file with O_CREATE and write data
		f, err = sfs.OpenFile(filename1, os.O_CREATE, 0o0644)
		require.NotNil(t, f)
		require.NoError(t, err)

		n, err = f.Write([]byte(value1))
		require.Equal(t, len(value1), n)
		require.NoError(t, err)

		require.NoError(t, f.Close())

		// open existing file with O_APPEND and write data
		f, err = sfs.OpenFile(filename1, os.O_APPEND, 0o0644)

		n, err = f.Write([]byte(value1))
		require.Equal(t, len(value1), n)
		require.NoError(t, err)

		require.NoError(t, f.Close())

		// read and check the written data
		f, err = sfs.Open(filename1)

		buf1 := make([]byte, 25)

		n, err = f.Read(buf1)
		require.Equal(t, 2*len(value1), n)
		require.NoError(t, err)
		require.Equal(t, value1+value1, string(buf1[:n]))

		require.NoError(t, f.Close())

		// open existing file with O_TRUNC and either O_RDWR or O_WRONLY and write data
		f, err = sfs.OpenFile(filename1, os.O_TRUNC|os.O_RDWR, 0o0644)

		n, err = f.Write([]byte(value1))
		require.Equal(t, len(value1), n)
		require.NoError(t, err)

		require.NoError(t, f.Close())

		// read and check the written data
		f, err = sfs.Open(filename1)

		buf2 := make([]byte, 25)

		n, err = f.Read(buf2)
		require.Equal(t, len(value1), n)
		require.NoError(t, err)
		require.Equal(t, value1, string(buf2[:n]))

		require.NoError(t, f.Close())

		// open existing file for writing
		f, err = sfs.OpenFile(filename1, os.O_RDWR, 0o0644)

		n, err = f.Write([]byte(value2))
		require.Equal(t, len(value2), n)
		require.NoError(t, err)

		require.NoError(t, f.Close())

		// read and check the written data
		f, err = sfs.Open(filename1)

		buf3 := make([]byte, 25)

		n, err = f.Read(buf3)
		require.Equal(t, len(value1), n)
		require.NoError(t, err)
		require.Equal(t, []byte("ABCDE56789"), buf3[:len(value1)])
	})
}

func TestFSRemove(t *testing.T) {
	sfs := secfs.New(backend.NewFakeClientset())
	require.NotNil(t, sfs)

	t.Run("Remove", func(t *testing.T) {
		secretname := "default/testsecret"
		filename := path.Join(secretname, "file1")

		err := sfs.Mkdir(secretname, os.FileMode(0))
		require.NoError(t, err)

		f, err := sfs.Create(filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		err = sfs.Remove(secretname)
		require.ErrorIs(t, err, syscall.ENOTEMPTY)

		err = sfs.Remove(filename)
		require.NoError(t, err)

		err = sfs.Remove(filename)
		require.ErrorIs(t, err, fs.ErrNotExist)

		f, err = sfs.Open(filename)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Nil(t, f)

		err = sfs.Remove(secretname)
		require.NoError(t, err)

		err = sfs.Remove(secretname)
		require.ErrorIs(t, err, fs.ErrNotExist)

		f, err = sfs.Open(secretname)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Nil(t, f)
	})

	t.Run("RemoveAll", func(t *testing.T) {
		secretname := "default/testsecret"
		filename1 := path.Join(secretname, "file1")
		filename2 := path.Join(secretname, "file2")

		err := sfs.Mkdir(secretname, os.FileMode(0))
		require.NoError(t, err)

		f, err := sfs.Create(filename1)
		require.NoError(t, err)
		require.NotNil(t, f)

		f, err = sfs.Create(filename2)
		require.NoError(t, err)
		require.NotNil(t, f)

		err = sfs.RemoveAll(filename1)
		require.NoError(t, err)

		f, err = sfs.Open(filename1)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Nil(t, f)

		err = sfs.RemoveAll(secretname)
		require.NoError(t, err)

		f, err = sfs.Open(secretname)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Nil(t, f)
	})
}

func TestFSRename(t *testing.T) {
	sfs := secfs.New(backend.NewFakeClientset())
	require.NotNil(t, sfs)

	t.Run("Rename with different namespace", func(t *testing.T) {
		secretname1 := "default/testsecret1"
		secretname2 := "scratch/testsecret1"

		err := sfs.Mkdir(secretname1, os.FileMode(0))
		require.NoError(t, err)

		err = sfs.Rename(secretname1, secretname2)
		require.ErrorIs(t, err, secfs.ErrMoveCrossNamespace)
	})

	t.Run("Rename secret", func(t *testing.T) {
		secretname1 := "default/testsecret2"
		secretname2 := "default/testsecret21"
		filename1 := "default/testsecret2/testfile"

		err := sfs.Rename(secretname1, secretname2)
		require.ErrorIs(t, err, fs.ErrNotExist, "%s should not exist", secretname1)

		err = sfs.Mkdir(secretname1, os.FileMode(0))
		require.NoError(t, err)

		err = sfs.Rename(secretname1, filename1)
		require.ErrorIs(t, err, secfs.ErrMoveConvert)

		err = sfs.Mkdir(secretname2, os.FileMode(0))
		require.NoError(t, err)

		err = sfs.Rename(secretname1, secretname2)
		require.ErrorIs(t, err, fs.ErrExist, "%s should already exist", secretname2)

		err = sfs.Remove(secretname2)
		require.NoError(t, err)

		err = sfs.Rename(secretname1, secretname2)
		require.NoError(t, err)

		f, err := sfs.Open(secretname1)
		require.ErrorIs(t, err, fs.ErrNotExist, "%s should no longer exist", secretname1)
		require.Nil(t, f)

		f, err = sfs.Open(secretname2)
		require.NoError(t, err)
		require.NotNil(t, f)
	})

	t.Run("Rename file", func(t *testing.T) {
		secretname1 := "default/testsecret3"
		filename11 := "default/testsecret3/testfile1"
		filename12 := "default/testsecret3/testfile2"

		err := sfs.Mkdir(secretname1, os.FileMode(0))
		require.NoError(t, err)

		err = sfs.Rename(filename11, filename12)
		require.ErrorIs(t, err, fs.ErrNotExist, "%s should not exist", filename11)

		f, err := sfs.Create(filename11)
		require.NoError(t, err)
		require.NotNil(t, f)

		f, err = sfs.Create(filename12)
		require.NoError(t, err)
		require.NotNil(t, f)

		// "If newpath already exists and is not a directory, Rename replaces it."
		// https://pkg.go.dev/os#Rename
		err = sfs.Rename(filename11, filename12)
		require.NoError(t, err)

		f, err = sfs.Open(filename11)
		require.ErrorIs(t, err, fs.ErrNotExist, "%s should no longer exist", filename11)
		require.Nil(t, f)

		f, err = sfs.Open(filename12)
		require.NoError(t, err)
		require.NotNil(t, f)
	})

	t.Run("Move file", func(t *testing.T) {
		secretname1 := "default/testsecret4"
		filename1 := "default/testsecret4/testfile1"

		secretname2 := "default/testsecret5"
		filename2 := "default/testsecret5/testfile1"

		err := sfs.Mkdir(secretname1, os.FileMode(0))
		require.NoError(t, err)

		f, err := sfs.Create(filename1)
		require.NoError(t, err)
		require.NotNil(t, f)

		err = sfs.Rename(filename1, secretname2)
		require.ErrorIs(t, err, fs.ErrNotExist, "%s should not exist", secretname2)

		err = sfs.Mkdir(secretname2, os.FileMode(0))
		require.NoError(t, err)

		f, err = sfs.Create(filename2)
		require.NoError(t, err)
		require.NotNil(t, f)

		// "If newpath already exists and is not a directory, Rename replaces it."
		// https://pkg.go.dev/os#Rename
		err = sfs.Rename(filename1, secretname2)
		require.NoError(t, err)

		f, err = sfs.Open(filename1)
		require.ErrorIs(t, err, fs.ErrNotExist, "%s should no longer exist", filename1)
		require.Nil(t, f)

		f, err = sfs.Open(filename2)
		require.NoError(t, err)
		require.NotNil(t, f)
	})
}
