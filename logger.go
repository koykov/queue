package queue

// Logger is an interface of logger interface.
// Prints verbose messages.
type Logger interface {
	Printf(format string, v ...any)
	Print(v ...any)
	Println(v ...any)
}
