package secretfs

import corev1 "k8s.io/api/core/v1"

const (
	AnnotationKey   = "secretfs"
	AnnotationValue = "v1"
)

func addAnnotation(s *corev1.Secret) {
	if s.Annotations == nil {
		s.Annotations = make(map[string]string)
	}

	s.Annotations[AnnotationKey] = AnnotationValue
}

func checkAnnotaion(s *corev1.Secret) bool {
	v, ok := s.Annotations[AnnotationKey]

	return ok && v == AnnotationValue
}
