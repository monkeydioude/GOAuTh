package tuple

type Tuple_3[A, B, C any] struct {
	A A
	B B
	C C
}

func Tuple3[A, B, C any](a A, b B, c C) Tuple_3[A, B, C] {
	return Tuple_3[A, B, C]{a, b, c}
}
