package stringUtil

import "strings"

func IsNotBlank(str string) bool {
	return len(strings.TrimSpace(str)) > 0
}
