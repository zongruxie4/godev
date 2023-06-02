package godev

import (
	"fmt"
	"net"
	"sync"
)

func (a *Args) TcpHandler(ln net.Listener, wg *sync.WaitGroup) {

	// Start a Goroutine to handle incoming TCP connections
	go a.handlerTcpMessages(ln, wg)
}

func (a *Args) handlerTcpMessages(ln net.Listener, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Failed to accept TCP connection: %s\n", err)
			continue
		}

		// Handle the incoming message
		go func(c net.Conn) {
			defer c.Close()

			// Leer los datos recibidos del Programa
			buf := make([]byte, 1024)
			n, err := c.Read(buf)
			if err != nil {
				fmt.Println("Error al leer los datos:", err)
				return
			}

			msg := string(buf[:n])

			switch msg {
			case "restart":

				// Stop the previous program
				a.StopProgram()

				// Start a new program
				a.StartProgram()

				// reload here idem

				a.Reload <- true
			case "reload":
				a.Reload <- true

			default:

				fmt.Println("mensaje desconocido recibido: ", msg)

			}

		}(conn)
	}
}
