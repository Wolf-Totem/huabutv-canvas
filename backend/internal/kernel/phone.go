package kernel

import (
	"regexp"
	"strings"
)

var chinaMobilePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

func NormalizeChinaMobile(value string) string {
	s := strings.TrimSpace(value)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "\u00a0", "")
	s = strings.TrimPrefix(s, "+86")
	s = strings.TrimPrefix(s, "86")
	return s
}

func IsChinaMobile(value string) bool {
	return chinaMobilePattern.MatchString(NormalizeChinaMobile(value))
}
