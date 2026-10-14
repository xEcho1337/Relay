package connection

import (
	"crypto/md5"
	"fmt"
	"net"
	"strings"

	"relay/config"
)

func resolveBackend(cfg *config.Config, host string) (string, error) {
	if i := strings.IndexByte(host, 0); i >= 0 {
		host = host[:i]
	}
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))

	if len(cfg.Servers) == 1 {
		for _, srv := range cfg.Servers {
			return srv.Address, nil
		}
	}

	for name, srv := range cfg.Servers {
		if strings.EqualFold(name, host) || (srv.ID != "" && strings.EqualFold(srv.ID, host)) {
			return srv.Address, nil
		}
	}

	return "", fmt.Errorf("no backend server was specified")
}

// offlineUUID generates the vanilla offline-mode UUID (v3 MD5 of
// "OfflinePlayer:" plus the username), with dashes.
func offlineUUID(name string) string {
	sum := md5.Sum([]byte("OfflinePlayer:" + name))
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

func clientIP(conn net.Conn) string {
	if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
		return addr.IP.String()
	}
	if host, _, err := net.SplitHostPort(conn.RemoteAddr().String()); err == nil {
		return host
	}
	return conn.RemoteAddr().String()
}
