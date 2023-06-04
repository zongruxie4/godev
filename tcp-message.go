package godev

import (
	"log"
	"net"
)

func sendTcpMessage(message string) {
	conn, err := net.Dial("tcp", "localhost:1234") // Dirección y puerto en los que el programa B está escuchando
	if err != nil {
		log.Println("Error Dial Tcp ", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte(message))
	if err != nil {
		log.Println("Error al escribir mensaje tcp ", message, err)
	}

}
