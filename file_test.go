package secfs_test

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"syscall"
	"testing"

	"github.com/marcsauter/secfs"
	"github.com/marcsauter/secfs/internal/backend"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestFileCreate(t *testing.T) {
	namespace := "default"
	secret := "testsecret"
	key := "testfile"

	filename := path.Join(namespace, secret, key)
	secretname := path.Join(namespace, secret)

	cs := backend.NewFakeClientset()
	b := backend.New(cs)

	// prepare
	sfs := secfs.New(cs)

	err := sfs.Mkdir(secretname, os.FileMode(0))
	require.NoError(t, err)

	t.Run("FileCreate on secret", func(t *testing.T) {
		f, err := secfs.FileCreate(b, secretname)
		require.ErrorIs(t, err, syscall.EISDIR)
		require.Nil(t, f)
	})

	t.Run("FileCreate on file", func(t *testing.T) {
		f, err := secfs.FileCreate(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface backend.Metadata
		require.Equal(t, namespace, f.Namespace())
		require.Equal(t, secret, f.Secret())
		require.Equal(t, key, f.Key())

		// interface os.FileInfo
		require.Equal(t, key, f.Name())
		require.Equal(t, int64(0), f.Size())
		require.Equal(t, fs.FileMode(0), f.Mode())
		require.False(t, f.ModTime().IsZero())
		require.False(t, f.IsDir())
		require.Equal(t, f, f.Sys())

		require.NoError(t, f.Close())
		require.ErrorIs(t, f.Close(), afero.ErrFileClosed)
	})
}

func TestFileOpen(t *testing.T) {
	namespace := "default"
	secret := "testsecret"
	key := "testfile"

	filename := path.Join(namespace, secret, key)
	secretname := path.Join(namespace, secret)

	cs := backend.NewFakeClientset()
	b := backend.New(cs)

	// prepare
	sfs := secfs.New(cs)

	err := sfs.Mkdir(secretname, os.FileMode(0))
	require.NoError(t, err)

	f, err := secfs.FileCreate(b, filename)
	require.NoError(t, err)
	require.NotNil(t, f)

	t.Run("Open secret", func(t *testing.T) {
		f, err := secfs.Open(b, secretname)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface backend.Metadata
		require.Equal(t, namespace, f.Namespace())
		require.Equal(t, secret, f.Secret())
		require.Empty(t, f.Key())

		// interface os.FileInfo
		require.Equal(t, secret, f.Name())
		require.Equal(t, int64(1), f.Size())
		require.Equal(t, fs.ModeDir, f.Mode())
		require.False(t, f.ModTime().IsZero())
		require.True(t, f.IsDir())
		require.Equal(t, f, f.Sys())

		require.NoError(t, f.Close())
		require.ErrorIs(t, f.Close(), afero.ErrFileClosed)
	})

	t.Run("Open file", func(t *testing.T) {
		f, err := secfs.Open(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		// interface backend.Metadata
		require.Equal(t, namespace, f.Namespace())
		require.Equal(t, secret, f.Secret())
		require.Equal(t, key, f.Key())

		// interface os.FileInfo
		require.Equal(t, key, f.Name())
		require.Equal(t, int64(0), f.Size())
		require.Equal(t, fs.FileMode(0), f.Mode())
		require.False(t, f.ModTime().IsZero())
		require.False(t, f.IsDir())
		require.Equal(t, f, f.Sys())

		require.NoError(t, f.Close())
		require.ErrorIs(t, f.Close(), afero.ErrFileClosed)
	})
}

func TestFileReadWriteSeek(t *testing.T) {
	namespace := "default"
	secret := "testsecret"
	key := "testfile"

	filename := path.Join(namespace, secret, key)
	secretname := path.Join(namespace, secret)

	cs := backend.NewFakeClientset()
	b := backend.New(cs)

	// prepare
	sfs := secfs.New(cs)

	err := sfs.Mkdir(secretname, os.FileMode(0))
	require.NoError(t, err)

	f, err := secfs.FileCreate(b, filename)
	require.NoError(t, err)
	require.NotNil(t, f)

	t.Run("Read secret", func(t *testing.T) {
		f, err := secfs.Open(b, secretname)
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err := f.Read([]byte{})
		require.Zero(t, n)
		require.ErrorIs(t, err, syscall.EISDIR)

		n, err = f.ReadAt([]byte{}, 10)
		require.Zero(t, n)
		require.ErrorIs(t, err, syscall.EISDIR)

		require.NoError(t, f.Close())
	})

	t.Run("Seek secret", func(t *testing.T) {
		f, err := secfs.Open(b, secretname)
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err := f.Seek(10, io.SeekStart)
		require.Zero(t, n)
		require.ErrorIs(t, err, syscall.EISDIR)

		require.NoError(t, f.Close())
	})

	t.Run("Write secret", func(t *testing.T) {
		f, err := secfs.Open(b, secretname)
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err := f.Write([]byte{})
		require.Zero(t, n)
		require.ErrorIs(t, err, syscall.EISDIR)

		n, err = f.WriteAt([]byte{}, 10)
		require.Zero(t, n)
		require.ErrorIs(t, err, syscall.EISDIR)

		n, err = f.WriteString("")
		require.Zero(t, n)
		require.ErrorIs(t, err, syscall.EISDIR)

		require.NoError(t, f.Close())
	})

	t.Run("Read file", func(t *testing.T) {
		f, err := secfs.Open(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err := f.Read([]byte{})
		require.Zero(t, n)
		require.ErrorIs(t, err, io.EOF)

		n, err = f.ReadAt([]byte{}, 10)
		require.Zero(t, n)
		require.ErrorIs(t, err, io.EOF)

		require.NoError(t, f.Close())

		n, err = f.Read([]byte{})
		require.Zero(t, n)
		require.ErrorIs(t, err, afero.ErrFileClosed)

		n, err = f.ReadAt([]byte{}, 10)
		require.Zero(t, n)
		require.ErrorIs(t, err, afero.ErrFileClosed)
	})

	t.Run("Seek file", func(t *testing.T) {
		f, err := secfs.Open(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err := f.Seek(10, io.SeekStart)
		require.Equal(t, int64(10), n)
		require.NoError(t, err)

		require.NoError(t, f.Close())

		n, err = f.Seek(10, io.SeekStart)
		require.Equal(t, int64(0), n)
		require.ErrorIs(t, err, afero.ErrFileClosed)
	})

	t.Run("Write file", func(t *testing.T) {
		const (
			value = "0123456789"
		)

		// open file read-only
		f, err := secfs.Open(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err := f.Write([]byte{})
		require.Zero(t, n)
		require.ErrorIs(t, err, syscall.EBADF)

		n, err = f.WriteAt([]byte{}, 10)
		require.Zero(t, n)
		require.ErrorIs(t, err, syscall.EBADF)

		// open file for writing
		f, err = secfs.FileCreate(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		size := len(value)

		n, err = f.Write([]byte(value))
		require.Equal(t, size, n)
		require.NoError(t, err)

		offset := 5

		n, err = f.WriteAt([]byte(value), int64(offset))
		require.Equal(t, size, n)
		require.NoError(t, err)

		require.NoError(t, f.Close())

		n, err = f.Write([]byte{})
		require.Zero(t, n)
		require.ErrorIs(t, err, afero.ErrFileClosed)

		n, err = f.WriteAt([]byte{}, int64(offset))
		require.Zero(t, n)
		require.ErrorIs(t, err, afero.ErrFileClosed)

		f, err = secfs.Open(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		sizeR := size + offset

		buf := make([]byte, 20)

		nR, err := f.Read(buf)
		require.Equal(t, sizeR, nR)
		require.NoError(t, err)
		require.Equal(t, []byte("012340123456789"), buf[:sizeR])
		require.Equal(t, []byte("012340123456789"), f.Value())
	})

	t.Run("WriteString", func(t *testing.T) {
		// open file read-only
		f, err := secfs.Open(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err := f.WriteString("")
		require.Zero(t, n)
		require.ErrorIs(t, err, syscall.EBADF)

		// open file for writing
		f, err = secfs.FileCreate(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err = f.WriteString("")
		require.Zero(t, n)
		require.NoError(t, err)

		require.NoError(t, f.Close())

		n, err = f.WriteString("")
		require.Zero(t, n)
		require.ErrorIs(t, err, afero.ErrFileClosed)
	})
}

func TestFileTruncate(t *testing.T) {
	namespace := "default"
	secret := "testsecret"
	key := "testfile"

	filename := path.Join(namespace, secret, key)
	secretname := path.Join(namespace, secret)

	cs := backend.NewFakeClientset()
	b := backend.New(cs)

	// prepare
	sfs := secfs.New(cs)

	err := sfs.Mkdir(secretname, os.FileMode(0))
	require.NoError(t, err)

	f, err := secfs.FileCreate(b, filename)
	require.NoError(t, err)
	require.NotNil(t, f)

	t.Run("Truncate", func(t *testing.T) {
		f, err := secfs.Open(b, secretname)
		require.NoError(t, err)
		require.NotNil(t, f)

		err = f.Truncate(10)
		require.ErrorIs(t, err, syscall.EISDIR)
	})

	t.Run("Truncate file", func(t *testing.T) {
		const (
			value = "0123456789"
		)
		// open file read-only
		f, err := secfs.Open(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		err = f.Truncate(int64(0))
		require.ErrorIs(t, err, syscall.EBADF)

		// open file for writing
		f, err = secfs.FileCreate(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err := f.Write([]byte(value))
		require.Equal(t, len(value), n)
		require.NoError(t, err)

		fi, err := f.Stat()
		require.NotNil(t, fi)
		require.NoError(t, err)
		require.Equal(t, int64(len(value)), fi.Size())

		require.NoError(t, f.Close())

		// 1st truncate
		truncSize1 := 10

		fw, err := sfs.OpenFile(filename, os.O_RDWR, 0o0600)
		require.NoError(t, fw.Truncate(int64(truncSize1)))
		require.NoError(t, fw.Close())

		// check 1st truncate
		f, err = secfs.Open(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		buf1 := make([]byte, 20)

		n1, err := f.Read(buf1)
		require.Equal(t, truncSize1, n1)
		require.NoError(t, err)

		fi, err = f.Stat()
		require.NotNil(t, fi)
		require.NoError(t, err)
		require.Equal(t, int64(truncSize1), fi.Size())

		require.NoError(t, f.Close())

		// 2nd truncate
		truncSize2 := 5

		fw, err = sfs.OpenFile(filename, os.O_RDWR, 0o0600)
		require.NoError(t, fw.Truncate(int64(truncSize2)))
		require.NoError(t, fw.Close())

		// check 2nd truncate
		f, err = secfs.Open(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		buf2 := make([]byte, 20)

		n2, err := f.Read(buf2)
		require.Equal(t, truncSize2, n2)
		require.NoError(t, err)

		fi, err = f.Stat()
		require.NotNil(t, fi)
		require.NoError(t, err)
		require.Equal(t, int64(truncSize2), fi.Size())

		require.NoError(t, f.Close())

		// truncate closed file
		err = f.Truncate(10)
		require.ErrorIs(t, err, afero.ErrFileClosed)
	})
}

func TestFileSync(t *testing.T) {
	namespace := "default"
	secret := "testsecret"
	key := "testfile"

	filename := path.Join(namespace, secret, key)
	secretname := path.Join(namespace, secret)

	cs := backend.NewFakeClientset()
	b := backend.New(cs)

	// prepare
	sfs := secfs.New(cs)

	err := sfs.Mkdir(secretname, os.FileMode(0))
	require.NoError(t, err)

	f, err := secfs.FileCreate(b, filename)
	require.NoError(t, err)
	require.NotNil(t, f)

	t.Run("Sync secret", func(t *testing.T) {
		f, err := secfs.Open(b, secretname)
		require.NoError(t, err)
		require.NotNil(t, f)

		err = f.Sync()
		require.ErrorIs(t, err, syscall.EISDIR)
	})

	t.Run("Sync file", func(t *testing.T) {
		// open file read-only
		f, err := secfs.Open(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)

		err = f.Sync()
		require.NoError(t, err)

		require.NoError(t, f.Close())

		err = f.Sync()
		require.ErrorIs(t, err, afero.ErrFileClosed)
	})

}

func TestFileReaddir(t *testing.T) {
	namespace := "default"
	secret := "testsecret3"
	key := "testfile3"

	secretname := path.Join(namespace, secret)

	cs := backend.NewFakeClientset()
	b := backend.New(cs)

	// prepare
	sfs := secfs.New(cs)

	err := sfs.Mkdir(secretname, os.FileMode(0))
	require.NoError(t, err)

	count := 10

	for i := 0; i < count; i++ {
		filename := path.Join(namespace, secret, fmt.Sprintf("%s%d", key, i))
		f, err := secfs.FileCreate(b, filename)
		require.NoError(t, err)
		require.NotNil(t, f)
	}

	t.Run("Readdir not dir", func(t *testing.T) {
		f, err := secfs.Open(b, path.Join(namespace, secret, key+"0"))
		require.NoError(t, err)
		require.NotNil(t, f)

		fi, err := f.Readdir(0)
		require.ErrorIs(t, err, syscall.ENOTDIR)
		require.Len(t, fi, 0)
	})

	t.Run("Readdir", func(t *testing.T) {
		f, err := secfs.Open(b, secretname)
		require.NoError(t, err)
		require.NotNil(t, f)

		fi, err := f.Readdir(count)
		require.NoError(t, err)
		require.Len(t, fi, count)

		n := make([]string, count)
		for i := range fi {
			n[i] = fi[i].Name()
		}

		sort.Strings(n)

		for i := 0; i < count; i++ {
			require.Equal(t, n[i], fmt.Sprintf("%s%d", key, i))
		}
	})

	t.Run("Readnames not dir", func(t *testing.T) {
		f, err := secfs.Open(b, path.Join(namespace, secret, key+"0"))
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err := f.Readdirnames(0)
		require.ErrorIs(t, err, syscall.ENOTDIR)
		require.Len(t, n, 0)
	})

	t.Run("Readnames", func(t *testing.T) {
		f, err := secfs.Open(b, secretname)
		require.NoError(t, err)
		require.NotNil(t, f)

		n, err := f.Readdirnames(count)
		require.NoError(t, err)
		require.Len(t, n, count)

		sort.Strings(n)

		for i := 0; i < count; i++ {
			require.Equal(t, n[i], fmt.Sprintf("%s%d", key, i))
		}
	})
}
