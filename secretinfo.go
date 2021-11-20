package secretfs

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
)

type secretInfo struct {
	name   string
	isDir  bool
	size   int64
	mtime  time.Time
	mode   fs.FileMode
	secret *corev1.Secret
}

var _ os.FileInfo = secretInfo{}

// Name returns the base name
func (si secretInfo) Name() string {
	return filepath.Base(si.name)
}

// Size returns length in bytes for keys
func (si secretInfo) Size() int64 {
	return si.size
}

// Mode returns file mode bits
func (si secretInfo) Mode() fs.FileMode {
	return si.mode
}

// Mode returns file mode bits
func (si secretInfo) ModTime() time.Time {
	return si.mtime
}

// IsDir returns true for a secret, false for a key
func (si secretInfo) IsDir() bool {
	return si.isDir
}

// Sys returns underlying data source (can return nil)
func (si secretInfo) Sys() interface{} {
	return si.secret
}
