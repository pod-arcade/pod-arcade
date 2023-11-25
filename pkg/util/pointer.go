package util

func TypeToPointer[T any](t T) *T {
	return &t
}
