package blqueue

// Logger is an interface of logger interface.
// Prints verbose messages.
type Logger interface {
	Printf(format string, v ...interface{})
	Print(v ...interface{})
	Println(v ...interface{})
}
