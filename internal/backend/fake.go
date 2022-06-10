package backend

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// NewFakeClientset returns a fake clientset for testing
func NewFakeClientset() kubernetes.Interface {
	return fake.NewSimpleClientset(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "notmanaged",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
	})
}
