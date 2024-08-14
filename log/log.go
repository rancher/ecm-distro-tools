package log

import "fmt"

type Logger struct {
	verbose bool
}

func NewLogger(verbose bool) Logger {
	return Logger{verbose: verbose}
}

func (l Logger) Print(a ...any) (int, error) {
	if l.verbose {
		return fmt.Print(a...)
	}
	return 0, nil
}

func (l Logger) Printf(format string, a ...any) (int, error) {
	if l.verbose {
		return fmt.Printf(format, a...)
	}
	return 0, nil
}

func (l Logger) Println(a ...any) (int, error) {
	if l.verbose {
		return fmt.Println(a...)
	}
	return 0, nil
}
