package servNameUtil

import (
	"strings"
)

func ExtractServName(host string) string {
	suffix := ".soa.proxy.net"
	if strings.HasSuffix(host, suffix) {
		host = strings.TrimSuffix(host, suffix)
	}
	return host
}
