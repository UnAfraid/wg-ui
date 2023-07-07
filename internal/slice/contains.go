package slice

func Contains[T comparable](values []T, v T) bool {
	for _, value := range values {
		if value == v {
			return true
		}
	}
	return false
}
