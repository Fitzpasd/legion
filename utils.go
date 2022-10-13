package main

import (
	"strconv"
	"strings"
)

func NormalizeIp(ip string) string {
	normalizedIp := ip

	if len(normalizedIp) == 4 {
		sb := new(strings.Builder)
		sb.WriteString(strconv.Itoa(int(ip[0])))
		sb.WriteRune('.')
		sb.WriteString(strconv.Itoa(int(ip[1])))
		sb.WriteRune('.')
		sb.WriteString(strconv.Itoa(int(ip[2])))
		sb.WriteRune('.')
		sb.WriteString(strconv.Itoa(int(ip[3])))

		normalizedIp = sb.String()
	}

	return normalizedIp
}
