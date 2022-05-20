package runners

import "strings"

func WhitespaceTrimRight(str string) string {
	return strings.TrimRight(str, "\n\r ")
}
