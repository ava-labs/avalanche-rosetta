package mapper

import "strings"

// EqualFoldContains checks if the array contains the string regardless of casing
func EqualFoldContains(arr []string, str string) bool {
	for _, a := range arr {
		if strings.EqualFold(a, str) {
			return true
		}
	}
	return false
}
