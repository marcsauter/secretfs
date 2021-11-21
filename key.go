package secretfs

import corev1 "k8s.io/api/core/v1"

func createKey(s *corev1.Secret, key string) {
	if s.Data == nil {
		s.Data = make(map[string][]byte)
	}

	s.Data[key] = []byte{}
}
