package slice

func MapVars[T any](slice []T, into ...*T) {
	ls := len(slice)
	li := len(into)
	for i := 0; i < ls && i < li; i++ {
		*into[i] = slice[i]
	}
}
