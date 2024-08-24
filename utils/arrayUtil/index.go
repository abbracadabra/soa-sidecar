package arrayUtil

func MapSlice[T any, R any](input []T, transform func(T) R) []R {
	output := make([]R, len(input))
	for i, v := range input {
		output[i] = transform(v)
	}
	return output
}
