package validator

import (
	"strings"
)

func ValidHostnamePort(s string) bool {
	sp := strings.Split(s, ":")
	if len(sp) != 2 {
		return false
	}
	if sp[0] == "" || sp[1] == "" {
		return false
	}
	return true
}
