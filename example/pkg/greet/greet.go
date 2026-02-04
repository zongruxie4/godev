package greet

import "github.com/tinywasm/fmt"

func Greet(target string) string {
	return fmt.Sprintf("Hello, %s 👋 from GO 5", target) // debug test
}
