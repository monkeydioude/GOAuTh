package option

type O[T any] struct {
	Some *T
}

func Some[T any](some *T) O[T] {
	return O[T]{
		Some: some,
	}
}

func None[T any]() O[T] {
	return O[T]{
		Some: nil,
	}
}

func (o O[T]) IsSome() bool {
	return o.Some != nil
}

func (o O[T]) IsNone() bool {
	return !o.IsSome()
}
