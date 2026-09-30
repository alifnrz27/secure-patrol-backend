package helper

import "fmt"

func StringToInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

func DerefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func Includes[T comparable](slice []T, value T) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
