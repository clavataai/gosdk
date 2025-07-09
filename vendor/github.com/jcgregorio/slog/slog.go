// slog defines a common interface for loggers.
package slog

// Logger is the interface that all loggers should conform to.
//
type Logger interface {
	// Fatal logs a fatal log and then exits the program.
	// Arguments are handled in the manner of fmt.Print.
	Fatal(args ...interface{})

	// Fatalf logs a fatal log and then exits the program.
	// Arguments are handled in the manner of fmt.Printf.
	Fatalf(format string, args ...interface{})

	// Error logs error logs.
	// Arguments are handled in the manner of fmt.Print.
	Error(args ...interface{})

	// Errorf logs error logs.
	// Arguments are handled in the manner of fmt.Printf.
	Errorf(format string, args ...interface{})

	// Warning logs warning logs.
	// Arguments are handled in the manner of fmt.Print.
	Warning(args ...interface{})

	// Warning logs warning logs.
	// Arguments are handled in the manner of fmt.Printf.
	Warningf(format string, args ...interface{})

	// Info logs informational logs.
	// Arguments are handled in the manner of fmt.Print.
	Info(args ...interface{})

	// Infof logs informational logs.
	// Arguments are handled in the manner of fmt.Printf.
	Infof(format string, args ...interface{})

	// Debug logs debugging logs.
	// Arguments are handled in the manner of fmt.Print.
	Debug(args ...interface{})

	// Debugf logs debugging logs.
	// Arguments are handled in the manner of fmt.Printf.
	Debugf(format string, args ...interface{})

	// Raw sends the string s to the logs without any additional formatting.
	Raw(s string)
}
