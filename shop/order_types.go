package shop

import (
	"strings"
)

const (
	OrderTypePrefixSize = 5
)

type OrderType string

func (t OrderType) Prefix() string {
	l := len(t)
	if l == 0 {
		return ""
	}

	var s = OrderTypePrefixSize
	if l <= s {
		s = l
	}

	return strings.ToUpper(string(t[:s]))
}
