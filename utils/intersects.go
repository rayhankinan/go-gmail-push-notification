package utils

func Intersects[T comparable](a, b []T) bool {
	set := make(map[T]struct{})

	for _, item := range a {
		set[item] = struct{}{}
	}

	isIntersecting := false
	for _, item := range b {
		if _, exists := set[item]; exists {
			isIntersecting = true
			break
		}
	}

	return isIntersecting
}
