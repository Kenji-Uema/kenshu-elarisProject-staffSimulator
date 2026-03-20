package helpers

type TestReporter interface {
	Helper()
	Fatalf(format string, args ...any)
}
