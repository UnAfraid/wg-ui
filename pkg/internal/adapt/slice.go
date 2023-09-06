package adapt

func Array[T, R any](items []T, adapterFn func(T) R) (elements []R) {
	if len(items) == 0 {
		return nil
	}

	elements = make([]R, 0, len(items))
	for _, item := range items {
		elements = append(elements, adapterFn(item))
	}
	return elements
}

func ArrayErr[T, R any](items []T, adapterFn func(T) (R, error)) (elements []R, err error) {
	if len(items) == 0 {
		return nil, nil
	}

	elements = make([]R, 0, len(items))
	for _, item := range items {
		element, err := adapterFn(item)
		if err != nil {
			return nil, err
		}
		elements = append(elements, element)
	}
	return elements, nil
}
