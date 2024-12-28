package main

import (
	"fmt"
	"time"
)

func Main() {
	totalTime := 30
	for i := 0; i < totalTime; i += 3 {
		fmt.Println("Hello, World!", i)
		time.Sleep(3 * time.Second)
	}
	fmt.Println("Finalizado")
}
