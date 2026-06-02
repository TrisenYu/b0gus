package misc_utils

// BoolToAnyInt has to explict declare which type the return value is.
// when given x is true, return 1. Otherwise, return 0.
func BoolToAnyInt[T integer](x bool) T {
	if x {
		return 1
	}
	return 0
}
