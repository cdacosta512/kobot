package logging

import (
	"fmt"

	"github.com/fatih/color"
)

// basic styles
var (
	infoColor    = color.New(color.FgCyan)
	successColor = color.New(color.FgGreen)
	warnColor    = color.New(color.FgYellow)
	errorColor   = color.New(color.FgRed)
	bold         = color.New(color.Bold)
	// runningColor = color.New(color.FgGreen)
	// kobot		 = color.New(color.FgHiBlack)
)

func Kobot(format string, a ...interface{}) string {
    tag := color.New(color.FgHiBlack).Sprint("SCAN INFO:")
    return fmt.Sprintf("       | %s %s", tag, fmt.Sprintf(format, a...))
}

func Action(msg string, args ...interface{}) {
	infoColor.Printf("RECOMMENDATION    ")
	fmt.Printf(msg+"\n", args...)
}

func Running(msg string, args ...interface{}) {
	bold.Printf("RUNNING    ")
	fmt.Printf(msg+"\n", args...)
}

func Info(msg string, args ...interface{}) {
	infoColor.Printf("INFO        ")
	fmt.Printf(msg+"\n", args...)
}

func Starting(msg string, args ...interface{}) {
	successColor.Printf("STARTING    ")
	fmt.Printf(msg+"\n", args...)
}

func Success(msg string, args ...interface{}) {
	successColor.Printf("OK          ")
	fmt.Printf(msg+"\n", args...)
}

func Warn(msg string, args ...interface{}) {
	warnColor.Printf("WARN        ")
	fmt.Printf(msg+"\n", args...)
}

func Error(msg string, args ...interface{}) {
	errorColor.Printf("ERROR       ")
	fmt.Printf(msg+"\n", args...)
}

func Title(msg string, args ...interface{}) {
	bold.Printf("\n%s\n", fmt.Sprintf(msg, args...))
}
