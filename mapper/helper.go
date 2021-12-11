package mapper

import "strings"

//ContainsNoCaseCheck checks if the array contains the string regardless of casing
func ContainsNoCaseCheck(arr []string, str string) bool {
	for _, a := range arr {
		if strings.ToLower(a) == strings.ToLower(str) {
			return true
		}
	}
	return false
}
