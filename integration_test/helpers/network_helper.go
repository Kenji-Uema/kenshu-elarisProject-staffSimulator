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
