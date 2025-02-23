package ptr

func Ptr[T any](value T) *T {
	return &value
}

func PtrNilOnEmpty[T any](value T) *T {
	switch t := any(value).(type) {
	case string:
		if t != "" {
			return &value
		}
	}
	return nil
}
