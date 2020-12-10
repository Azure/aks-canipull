package utils

import (
	"strings"
)

// LocationEquals returns if the locations are same
func LocationEquals(l1, l2 string) bool {
	n1 := strings.ReplaceAll(l1, " ", "")
	n2 := strings.ReplaceAll(l2, " ", "")

	return strings.EqualFold(n1, n2)
}
