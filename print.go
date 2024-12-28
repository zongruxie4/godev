package godev

import (
	"fmt"
)

const format_color = "\033[%sm%s\033[0m"

// options: ok,err,warn,info
func print(message string, options ...string) {
	if format_color == "" {
		fmt.Print(message)
	} else {

		var color string

		for _, opt := range options {
			switch opt {
			case "ok":
				color = "32" //green
			case "err":
				color = "31" //red
			case "warn":
				color = "33" //yellow
			case "info":
				color = "36" //magenta blue=34
			default:
				color = "0"
			}
		}

		fmt.Printf(format_color, color, message)
	}
}

func PrintOK(messages ...any) {
	print(joinMessages(messages...), "ok")
}

func PrintWarning(messages ...any) {
	print(joinMessages(messages...), "warn")
}

func PrintError(messages ...any) {
	print(joinMessages(messages...), "err")
}

func PrintInfo(messages ...any) {
	print(joinMessages(messages...), "info")
}

func joinMessages(messages ...any) (message string) {
	var space string
	for _, m := range messages {
		message += space + fmt.Sprint(m)
		space = " "
	}
	return
}
