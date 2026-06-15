package registry

import (
	"strings"
)

func blank(value string) bool {
	return strings.TrimSpace(value) == ""
}
