package v1

import (
	"net"
	"net/url"
	"strings"
)

// privateRanges содержит приватные подсети по RFC 1918 и loopback.
var privateRanges = []net.IPNet{
	parseCIDR("127.0.0.0/8"),    // loopback IPv4
	parseCIDR("::1/128"),        // loopback IPv6
	parseCIDR("10.0.0.0/8"),     // Class A private
	parseCIDR("172.16.0.0/12"),  // Class B private
	parseCIDR("192.168.0.0/16"), // Class C private
}

func parseCIDR(s string) net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic("invalid CIDR: " + s)
	}
	return *network
}

func isPrivateIP(ipStr string) bool {
	// Убираем зону IPv6 (например, "::1%lo0")
	if idx := strings.IndexByte(ipStr, '%'); idx != -1 {
		ipStr = ipStr[:idx]
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

// isLocalNetworkOrigin разрешает запросы только с localhost и устройств
// в локальной сети (RFC 1918: 10.x, 172.16-31.x, 192.168.x).
func isLocalNetworkOrigin(origin string) bool {
	if origin == "" {
		return true // server-to-server или curl без Origin
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := u.Hostname() // убирает порт

	// Явные алиасы localhost
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}

	return isPrivateIP(host)
}
