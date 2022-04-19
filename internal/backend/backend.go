// Package backend provides CRUD for the secrets
package backend

import (
	"fmt"
	"time"

	"github.com/marcsauter/sekretsfs/internal/secret"
	"github.com/spf13/afero"
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
	// AnnotationKey is the name of the sekretsfs annotation
	AnnotationKey = "sekretsfs"
	// AnnotationValue is the sekretsfs version
	AnnotationValue = "v1"
)

// Backend implements the communication with Kubernetes
type Backend struct {
	c          kubernetes.Interface
	secretType corev1.SecretType
	timeout    time.Duration
	l          *zap.SugaredLogger
}

// New returns a Backend
func New(c kubernetes.Interface, opts ...Option) *Backend {
	b := &Backend{
		c:       c,
		timeout: DefaultRequestTimeout,
	}

	for _, option := range opts {
		option(b)
	}

	return b
}

// Load secret from backend
func (b *Backend) Load(s *secret.Secret) error {
	ks, err := b.get(s)

	if apierr.IsNotFound(err) {
		return afero.ErrFileNotFound
	}

	if err != nil {
		return err
	}

	s.SetData(ks.Data)

	return nil
}

// Store secret in backend
func (b *Backend) Store(s *secret.Secret) error {
	ks, err := b.get(s)

	ks.Data = s.Data()

	if apierr.IsNotFound(err) {
		ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
		defer cancel()

		_, err = b.c.CoreV1().Secrets(s.Namespace()).Create(ctx, ks, metav1.CreateOptions{})

		return err
	}

	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	_, err = b.c.CoreV1().Secrets(s.Namespace()).Update(ctx, ks, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// Delete secret in backend
func (b *Backend) Delete(s *secret.Secret) error {
	_, err := b.get(s)
	if apierr.IsNotFound(err) {
		return nil
	}

	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	if err := b.c.CoreV1().Secrets(s.Namespace()).Delete(ctx, s.Secret(), metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func (b *Backend) get(s *secret.Secret) (*corev1.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	ks, err := b.c.CoreV1().Secrets(s.Namespace()).Get(ctx, s.Secret(), metav1.GetOptions{})
	if apierr.IsNotFound(err) {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: s.Name(),
				Annotations: map[string]string{
					AnnotationKey: AnnotationValue,
				},
			},
		}, err
	}

	if err != nil {
		return nil, err
	}

	if v, ok := ks.Annotations[AnnotationKey]; ok && v == AnnotationValue {
		return ks, nil
	}

	return nil, fmt.Errorf("not managed with sekretsfs")
}
