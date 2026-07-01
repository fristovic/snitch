//go:build !windows

package ipc

import (
	"net"
	"os"
	"path/filepath"
)

func listen(addr string) (net.Listener, error) {
	if len(addr) > 0 && addr[0] != '\\' {
		_ = os.Remove(addr)
		_ = os.MkdirAll(filepath.Dir(addr), 0o700)
	}
	ln, err := net.Listen("unix", addr)
	if err != nil {
		return nil, err
	}
	_ = os.Chmod(addr, 0o600)
	return ln, nil
}

func dial(addr string) (net.Conn, error) {
	return net.Dial("unix", addr)
}
