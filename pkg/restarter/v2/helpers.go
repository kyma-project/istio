package v2

// ptrEqual returns true if two comparable type pointers are equal.
func ptrEqual[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
