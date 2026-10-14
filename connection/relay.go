package connection

import "io"

import "net"

func relay(a, b net.Conn) {
	go func() {
		_, _ = io.Copy(b, a)
		_ = a.Close()
		_ = b.Close()
	}()
	_, _ = io.Copy(a, b)
	_ = a.Close()
	_ = b.Close()
}
