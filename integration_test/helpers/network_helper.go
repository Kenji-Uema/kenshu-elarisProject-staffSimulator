package helpers

import (
	"net"
)

func FreeTCPPort(t TestReporter) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on ephemeral port: %v", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port
}

func HostOnly(t TestReporter, addr string) string {
	t.Helper()

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split host from %q: %v", addr, err)
	}
	return host
}

func PortOnly(t TestReporter, addr string) string {
	t.Helper()

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split port from %q: %v", addr, err)
	}
	return port
}
