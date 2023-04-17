package adapt

func ToPointer[T any](v T) *T {
	return &v
}

func ToPointerNilZero[T comparable](v T) *T {
	var empty T
	if v == empty {
		return nil
	}
	return &v
}

func Dereference[T any](v *T) (result T) {
	if v == nil {
		return result
	}
	return *v
}
