package database

import "fmt"

var counter = 1

func Connect() {

	fmt.Println("Conn. to the database 1...", counter)

	counter++

}
