package backend

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

// Constants for testing with fake backend
const (
	FakePrefix = "unit-"
	FakeSuffix = "-test"
)

// NewFakeClientset returns a fake clientset for testing
func NewFakeClientset() kubernetes.Interface {
	return fake.NewSimpleClientset(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%snotmanaged%s", FakePrefix, FakeSuffix),
			Namespace:   "default",
			Annotations: map[string]string{},
		},
	})
}
