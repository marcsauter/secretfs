// Package backend provides CRUD for the secrets
package backend

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/context"

	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// DefaultRequestTimeout for k8s requests
	DefaultRequestTimeout = 5 * time.Second
	// AnnotationKey is the name of the secfs annotation
	AnnotationKey = "secfs"
	// AnnotationValue is the secfs version
	AnnotationValue = "v1"
	// ModTimeKey is the name of the modification time annotation
	ModTimeKey = "modtime"
)

var (
	// ErrNotManaged for secrets not managed with secfs
	ErrNotManaged = errors.New("not managed with secfs")
)

// Metadata is the interface for basic metadata information
type Metadata interface {
	Namespace() string
	Secret() string
	Key() string
}

// Secret is the interface that abstracts the Kubernetes secret
type Secret interface {
	Metadata

	Value() []byte
	Delete() bool // delete Key() from map

	Data() map[string][]byte
	SetData(map[string][]byte)

	SetTime(time.Time)
}

// Backend is the interface that groups the basic Create, Get, Update and Delete methods.
type Backend interface {
	Create(Secret) error
	Get(Secret) error
	Update(Secret) error
	Delete(Secret) error
	Rename(Metadata, Metadata) error
}

// backend implements the communication with Kubernetes
type backend struct {
	c      kubernetes.Interface
	prefix string
	suffix string
	labels map[string]string

	ignoreAnnotation bool

	mu      sync.Mutex
	timeout time.Duration
	l       *zap.SugaredLogger
}

// New returns a Backend
func New(c kubernetes.Interface, opts ...Option) Backend {
	b := &backend{
		c:       c,
		timeout: DefaultRequestTimeout,
	}

	for _, option := range opts {
		option(b)
	}

	return b
}

// Create secret in backend
func (b *backend) Create(s Secret) error {
	ks := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   b.internalName(s.Secret()),
			Labels: b.labels,
			Annotations: map[string]string{
				AnnotationKey: AnnotationValue,
			},
		},
		Data: s.Data(),
	}

	setCurrentTime(ks)

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	_, err := b.c.CoreV1().Secrets(s.Namespace()).Create(ctx, ks, metav1.CreateOptions{})

	return err
}

// Get secret from backend
func (b *backend) Get(s Secret) error {
	ks, err := b.get(s)

	// map error
	if apierr.IsNotFound(err) {
		return syscall.ENOENT
	}

	if err != nil {
		return err
	}

	s.SetData(ks.Data)
	s.SetTime(getTime(ks))

	return nil
}

// Update secret in backend
func (b *backend) Update(s Secret) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	ks, err := b.get(s)
	if err != nil {
		return err
	}

	if s.Delete() {
		delete(ks.Data, s.Key())
	} else {
		ks.Data[s.Key()] = s.Value()
	}

	setCurrentTime(ks)
	s.SetTime(getTime(ks))

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	_, err = b.c.CoreV1().Secrets(s.Namespace()).Update(ctx, ks, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// Delete secret in backend
func (b *backend) Delete(s Secret) error {
	_, err := b.get(s)

	if apierr.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	if err := b.c.CoreV1().Secrets(s.Namespace()).Delete(ctx, b.internalName(s.Secret()), metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

// Rename secret in backend
func (b *backend) Rename(o, n Metadata) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	s, err := b.get(o)
	// source not found
	if apierr.IsNotFound(err) {
		return syscall.ENOENT
	}
	// backend error
	if err != nil {
		return err
	}

	_, err = b.get(n)
	// target already exists
	if err == nil {
		return syscall.EEXIST
	}
	// backend error
	if !apierr.IsNotFound(err) {
		return err
	}

	// rename
	s.Name = b.internalName(n.Secret())
	setCurrentTime(s)

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	// create new secret
	if _, err := b.c.CoreV1().Secrets(n.Namespace()).Create(ctx, s, metav1.CreateOptions{}); err != nil {
		return err
	}

	// delete old secret
	if err := b.c.CoreV1().Secrets(o.Namespace()).Delete(ctx, b.internalName(o.Secret()), metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func (b *backend) get(s Metadata) (*corev1.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	ks, err := b.c.CoreV1().Secrets(s.Namespace()).Get(ctx, b.internalName(s.Secret()), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if ks.Data == nil {
		ks.Data = make(map[string][]byte)
	}

	if !b.checkAnnotation(ks) {
		return nil, ErrNotManaged
	}

	return ks, nil
}

// internal

// internalName is the name of the secret in the backend
func (b *backend) internalName(name string) string {
	return fmt.Sprintf("%s%s%s", b.prefix, name, b.suffix)
}

// externalName is the name of the secret used in path
//
//nolint:unused // logical opposite to internalName
func (b *backend) externalName(name string) string {
	return strings.TrimSuffix(strings.TrimPrefix(name, b.prefix), b.suffix)
}

// checkAnnotation checks if the correct annotation for secfs is set
// if ignoreAnnotation is set to true, the annotation will not be checked
func (b *backend) checkAnnotation(ks *corev1.Secret) bool {
	if b.ignoreAnnotation {
		return true
	}

	v, ok := ks.Annotations[AnnotationKey]

	return ok && v == AnnotationValue
}

// helpers

func setCurrentTime(s *corev1.Secret) {
	s.Annotations[ModTimeKey] = time.Now().Format(time.RFC3339)
}

func getTime(s *corev1.Secret) time.Time {
	t, err := time.Parse(time.RFC3339, s.Annotations[ModTimeKey])
	if err != nil {
		return time.Now()
	}

	return t
}
