package log

import "fmt"

type Logger struct {
	verbose bool
}

func NewLogger(verbose bool) Logger {
	return Logger{verbose: verbose}
}

func (l Logger) Print(a ...any) (n int, err error) {
	if l.verbose {
		return fmt.Print(a...)
	}
	return
}

func (l Logger) Printf(format string, a ...any) (n int, err error) {
	if l.verbose {
		return fmt.Printf(format, a...)
	}
	return
}

func (l Logger) Println(a ...any) (n int, err error) {
	if l.verbose {
		return fmt.Println(a...)
	}
	return
}
